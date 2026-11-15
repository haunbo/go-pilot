package xfile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteReadAndStat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "config.json")
	if err := WriteFileAtomic(path, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if !Exists(path) {
		t.Fatal("file should exist")
	}
	sum, err := MD5(path)
	if err != nil || len(sum) != 32 {
		t.Fatalf("MD5 = %q, %v", sum, err)
	}
	info, err := Stat(path)
	if err != nil || info.Size == 0 {
		t.Fatalf("Stat = %+v, %v", info, err)
	}
}

func TestListAndRecursiveFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.json"), []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := ListFiles(dir, ".json")
	if err != nil || len(files) != 1 {
		t.Fatalf("ListFiles = %v, %v", files, err)
	}
	files, err = RecursiveFiles(dir, "*.json")
	if err != nil || len(files) != 1 {
		t.Fatalf("RecursiveFiles = %v, %v", files, err)
	}
}

func TestIsFileChangedAndCopy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := os.WriteFile(src, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	changed, err := IsFileChanged(src, dst)
	if err != nil || !changed {
		t.Fatalf("IsFileChanged missing dest = %v, %v", changed, err)
	}
	if _, err := Copy(src, dst); err != nil {
		t.Fatal(err)
	}
	changed, err = IsFileChanged(src, dst)
	if err != nil || changed {
		t.Fatalf("IsFileChanged after copy = %v, %v", changed, err)
	}
	if err := os.WriteFile(dst, []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	changed, _ = IsFileChanged(src, dst)
	if !changed {
		t.Fatal("content change should be detected")
	}
}

func TestLock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.lock")
	first, err := Lock(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Lock(path, false); !errors.Is(err, ErrLocked) {
		t.Fatalf("second lock error = %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := Lock(path, false)
	if err != nil {
		t.Fatalf("lock after release: %v", err)
	}
	_ = second.Close()
}
