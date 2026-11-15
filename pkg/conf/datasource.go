package conf

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// DataSource supplies raw configuration content and optionally change events.
type DataSource interface {
	ReadConfig() ([]byte, error)
	// IsConfigChanged returns nil when the data source cannot be watched.
	IsConfigChanged() <-chan struct{}
	io.Closer
}

type fileDataSource struct {
	path      string
	changed   chan struct{}
	done      chan struct{}
	watcher   *fsnotify.Watcher
	closeOnce sync.Once
}

// NewFileDataSource returns a file backed data source. When watch is true the
// parent directory is watched and changes to path are debounced.
func NewFileDataSource(path string, watch bool) (DataSource, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, fmt.Errorf("conf: stat %s: %w", abs, err)
	}
	ds := &fileDataSource{path: abs, done: make(chan struct{})}
	if !watch {
		return ds, nil
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("conf: create file watcher: %w", err)
	}
	if err := w.Add(filepath.Dir(abs)); err != nil {
		_ = w.Close()
		return nil, fmt.Errorf("conf: watch dir %s: %w", filepath.Dir(abs), err)
	}
	ds.watcher = w
	ds.changed = make(chan struct{}, 1)
	go ds.watch()
	return ds, nil
}

// ReadConfig reads the whole file.
func (f *fileDataSource) ReadConfig() ([]byte, error) {
	return os.ReadFile(f.path)
}

// IsConfigChanged returns the change notification channel, or nil without watch.
func (f *fileDataSource) IsConfigChanged() <-chan struct{} {
	return f.changed
}

func (f *fileDataSource) watch() {
	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		select {
		case <-f.done:
			return
		case event, ok := <-f.watcher.Events:
			if !ok {
				return
			}
			if filepath.Clean(event.Name) != filepath.Clean(f.path) {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(200*time.Millisecond, func() {
				select {
				case f.changed <- struct{}{}:
				default:
				}
			})
		case err, ok := <-f.watcher.Errors:
			if !ok {
				return
			}
			_ = err // watcher errors are non-fatal; the next change retries
		}
	}
}

// Close stops watching the file.
func (f *fileDataSource) Close() error {
	var err error
	f.closeOnce.Do(func() {
		close(f.done)
		if f.watcher != nil {
			err = f.watcher.Close()
		}
	})
	return err
}
