// Package xfile contains filesystem helpers used by backend services.
package xfile

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

// ErrLocked is returned when a non-blocking lock cannot be acquired.
var ErrLocked = errors.New("xfile: file is already locked")

// FileInfo describes a regular file.
type FileInfo struct {
	Size    int64
	Mode    os.FileMode
	ModTime time.Time
	MD5     string
}

// Exists reports whether path exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDir reports whether path is a directory.
func IsDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// EnsureDir creates all directories if they do not exist.
func EnsureDir(dirs ...string) error {
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("xfile: create dir %s: %w", dir, err)
		}
	}
	return nil
}

// ListFiles returns regular files in dir filtered by extension. ext may be
// ".json" or "json"; an empty ext returns all files.
func ListFiles(dir, ext string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	ext = normalizeExt(ext)
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if ext != "" && !strings.EqualFold(filepath.Ext(entry.Name()), ext) {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	return files, nil
}

func normalizeExt(ext string) string {
	if ext == "" {
		return ""
	}
	if !strings.HasPrefix(ext, ".") {
		return "." + ext
	}
	return ext
}

// RecursiveFiles walks root and returns files whose base name matches pattern.
// If root is a regular file it is returned when it matches.
func RecursiveFiles(root, pattern string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		match, err := filepath.Match(pattern, filepath.Base(root))
		if err != nil {
			return nil, err
		}
		if match {
			return []string{root}, nil
		}
		return []string{}, nil
	}

	files := make([]string, 0)
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		match, err := filepath.Match(pattern, entry.Name())
		if err != nil {
			return err
		}
		if match {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// MD5 returns the lowercase hex MD5 digest of a file's contents.
func MD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Stat returns metadata and the content hash of path.
func Stat(path string) (FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, err
	}
	sum, err := MD5(path)
	if err != nil {
		return FileInfo{}, err
	}
	return FileInfo{Size: info.Size(), Mode: info.Mode(), ModTime: info.ModTime(), MD5: sum}, nil
}

// IsFileChanged reports whether two files differ in mode or content.
func IsFileChanged(src, dest string) (bool, error) {
	if !Exists(dest) {
		return true, nil
	}
	srcInfo, err := Stat(src)
	if err != nil {
		return true, err
	}
	destInfo, err := Stat(dest)
	if err != nil {
		return true, err
	}
	return srcInfo.Mode != destInfo.Mode || srcInfo.MD5 != destInfo.MD5, nil
}

// Copy copies src to dst, creating parent directories and preserving the
// source file mode.
func Copy(src, dst string) (int64, error) {
	in, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return 0, err
	}
	if err := EnsureDir(filepath.Dir(dst)); err != nil {
		return 0, err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return 0, err
	}
	written, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return written, copyErr
	}
	if closeErr != nil {
		return written, closeErr
	}
	return written, nil
}

// WriteFileAtomic writes data to a temporary file in the same directory and
// renames it into place.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// Lock acquires an advisory file lock. When wait is false it returns ErrLocked
// immediately if the lock is held elsewhere.
func Lock(path string, wait bool) (io.Closer, error) {
	l := flock.New(path)
	if wait {
		if err := l.Lock(); err != nil {
			_ = l.Close()
			return nil, err
		}
		return lockCloser{l}, nil
	}
	locked, err := l.TryLock()
	if err != nil {
		_ = l.Close()
		return nil, err
	}
	if !locked {
		_ = l.Close()
		return nil, ErrLocked
	}
	return lockCloser{l}, nil
}

type lockCloser struct {
	l *flock.Flock
}

func (l lockCloser) Close() error {
	unlockErr := l.l.Unlock()
	closeErr := l.l.Close()
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}
