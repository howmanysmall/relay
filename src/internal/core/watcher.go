package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher monitors file system changes with debouncing.
type FileWatcher struct {
	watcher   *fsnotify.Watcher
	events    chan ChangeEvent
	errors    chan error
	debouncer *eventDebouncer
	mu        sync.RWMutex
	watching  map[string]bool
	running   bool
}

type eventDebouncer struct {
	delay   time.Duration
	pending map[string]*time.Timer
	mu      sync.Mutex
}

// NewFileWatcher creates a new file watcher with the specified debounce delay.
func NewFileWatcher(debounceDelay time.Duration) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	if debounceDelay == 0 {
		debounceDelay = 100 * time.Millisecond
	}

	return &FileWatcher{
		watcher:  watcher,
		events:   make(chan ChangeEvent, 1000),
		errors:   make(chan error, 100),
		watching: make(map[string]bool),
		debouncer: &eventDebouncer{
			delay:   debounceDelay,
			pending: make(map[string]*time.Timer),
		},
	}, nil
}

// Start begins monitoring file system events.
func (fw *FileWatcher) Start(ctx context.Context) error {
	fw.mu.Lock()

	if fw.running {
		fw.mu.Unlock()
		return fmt.Errorf("watcher is already running")
	}

	fw.running = true
	fw.mu.Unlock()

	go fw.eventLoop(ctx)

	return nil
}

// Stop stops monitoring file system events.
func (fw *FileWatcher) Stop() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if !fw.running {
		return fmt.Errorf("watcher is not running")
	}

	fw.running = false

	if err := fw.watcher.Close(); err != nil {
		return fmt.Errorf("failed to close watcher: %w", err)
	}

	close(fw.events)
	close(fw.errors)

	fw.debouncer.mu.Lock()

	for _, timer := range fw.debouncer.pending {
		timer.Stop()
	}

	fw.debouncer.pending = make(map[string]*time.Timer)
	fw.debouncer.mu.Unlock()

	return nil
}

// Add adds a path to be monitored for changes.
func (fw *FileWatcher) Add(path string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", path, err)
	}

	if fw.watching[absPath] {
		return nil
	}

	if err := fw.watcher.Add(absPath); err != nil {
		return fmt.Errorf("failed to watch path %s: %w", absPath, err)
	}

	fw.watching[absPath] = true

	return nil
}

// Remove stops monitoring the specified path.
func (fw *FileWatcher) Remove(path string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", path, err)
	}

	if !fw.watching[absPath] {
		return nil
	}

	if err := fw.watcher.Remove(absPath); err != nil {
		return fmt.Errorf("failed to stop watching path %s: %w", absPath, err)
	}

	delete(fw.watching, absPath)

	return nil
}

// Events returns the channel for receiving file change events.
func (fw *FileWatcher) Events() <-chan ChangeEvent {
	return fw.events
}

// Errors returns the channel for receiving watcher errors.
func (fw *FileWatcher) Errors() <-chan error {
	return fw.errors
}

func (fw *FileWatcher) eventLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			fw.handleEvent(event)

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}

			select {
			case fw.errors <- err:
			default:
			}
		}
	}
}

func (fw *FileWatcher) handleEvent(event fsnotify.Event) {
	changeType := fw.mapEventType(event.Op)

	changeEvent := ChangeEvent{
		Type:      changeType,
		Path:      event.Name,
		Timestamp: time.Now(),
	}

	fw.debouncer.debounce(event.Name, func() {
		scanner := NewFileScanner(1)
		if info, err := scanner.getFileInfoFromPath(event.Name); err == nil {
			changeEvent.Info = info
		}

		select {
		case fw.events <- changeEvent:
		default:
		}
	})
}

func (fw *FileWatcher) mapEventType(op fsnotify.Op) ChangeType {
	switch {
	case op&fsnotify.Create == fsnotify.Create:
		return ChangeCreate
	case op&fsnotify.Write == fsnotify.Write:
		return ChangeModify
	case op&fsnotify.Remove == fsnotify.Remove:
		return ChangeDelete
	case op&fsnotify.Rename == fsnotify.Rename:
		return ChangeRename
	default:
		return ChangeModify
	}
}

func (d *eventDebouncer) debounce(key string, fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if timer, exists := d.pending[key]; exists {
		timer.Stop()
	}

	d.pending[key] = time.AfterFunc(d.delay, func() {
		d.mu.Lock()
		delete(d.pending, key)
		d.mu.Unlock()
		fn()
	})
}

func (s *FileScanner) getFileInfoFromPath(path string) (*FileInfo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %s: %w", path, err)
	}

	info := &FileInfo{
		Path:    path,
		Size:    stat.Size(),
		ModTime: stat.ModTime(),
		Mode:    uint32(stat.Mode()),
		IsDir:   stat.IsDir(),
	}

	if !info.IsDir && info.Size > 0 {
		if checksum, err := s.getChecksum(path, info); err == nil {
			info.Checksum = checksum
			info.ChecksumAlgo = s.checksumAlgo
		}
	}

	return info, nil
}
