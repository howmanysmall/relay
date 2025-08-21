package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/howmanysmall/relay/src/internal/config"
)

// SyncEngine orchestrates file synchronization operations.
type SyncEngine struct {
	scanner      *FileScanner
	copier       *FileCopier
	watcher      *FileWatcher
	resolver     *ConflictResolver
	retryManager *RetryManager
	errorHandler *ErrorHandler
	stats        *SyncStats
	progress     *Progress
	mu           sync.RWMutex
}

// NewSyncEngine creates a new synchronization engine with default settings.
func NewSyncEngine() (*SyncEngine, error) {
	watcher, err := NewFileWatcher(100 * time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	return &SyncEngine{
		scanner:      NewFileScanner(0),      // Auto-detect concurrency
		copier:       NewFileCopier(0, true), // Auto buffer size, enable zero-copy
		watcher:      watcher,
		resolver:     NewConflictResolver(nil), // Use default config
		retryManager: NewRetryManager(nil),     // Use default retry config
		errorHandler: NewErrorHandler(1000),    // Max 1000 errors
		stats:        &SyncStats{},
		progress:     &Progress{},
	}, nil
}

// Mirror performs one-way mirroring from source to destination.
func (e *SyncEngine) Mirror(ctx context.Context, source, destination string) error {
	e.resetStats()
	e.stats.StartTime = time.Now()

	opts := SyncOptions{
		DryRun:           false,
		Recursive:        true,
		PreservePerms:    true,
		PreserveTimes:    true,
		DeleteExtraneous: false,
		ChecksumVerify:   true,
		Workers:          0, // Auto-detect
	}

	_, err := e.Sync(ctx, source, destination, opts)

	return err
}

// Sync performs synchronization between source and destination with the given options.
func (e *SyncEngine) Sync(ctx context.Context, source, destination string, opts SyncOptions) (*SyncStats, error) {
	e.resetStats()
	e.stats.StartTime = time.Now()

	sourceFiles, err := e.scanner.Scan(ctx, source)
	if err != nil {
		return e.stats, fmt.Errorf("failed to scan source directory: %w", err)
	}

	e.stats.FilesScanned = int64(len(sourceFiles))
	e.progress.Total = int64(len(sourceFiles))

	destFiles, err := e.scanner.Scan(ctx, destination)
	if err != nil {
		if !os.IsNotExist(err) {
			return e.stats, fmt.Errorf("failed to scan destination directory: %w", err)
		}

		destFiles = []*FileInfo{}
	}

	destMap := make(map[string]*FileInfo)

	for _, file := range destFiles {
		relPath, _ := filepath.Rel(destination, file.Path)
		destMap[relPath] = file
	}

	var wg sync.WaitGroup

	semaphore := make(chan struct{}, opts.Workers)
	if opts.Workers <= 0 {
		semaphore = make(chan struct{}, e.scanner.maxConcurrency)
	}

	for i, sourceFile := range sourceFiles {
		select {
		case <-ctx.Done():
			return e.stats, ctx.Err()
		case semaphore <- struct{}{}:
		}

		wg.Add(1)

		go func(file *FileInfo, _ int) {
			defer func() {
				<-semaphore
				wg.Done()
				atomic.AddInt64(&e.progress.Current, 1)
				e.updateProgress(file.Path)
			}()

			if err := e.syncFile(ctx, source, destination, file, destMap, opts); err != nil {
				atomic.AddInt64(&e.stats.ErrorsEncountered, 1)
			}
		}(sourceFile, i)
	}

	wg.Wait()

	e.stats.EndTime = time.Now()
	e.stats.Duration = e.stats.EndTime.Sub(e.stats.StartTime)

	return e.stats, nil
}

func (e *SyncEngine) syncFile(ctx context.Context, source, destination string, sourceFile *FileInfo, destMap map[string]*FileInfo, opts SyncOptions) error {
	relPath, err := filepath.Rel(source, sourceFile.Path)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	destPath := filepath.Join(destination, relPath)
	destFile, exists := destMap[relPath]

	needsSync := !exists
	if exists {
		needsSync = e.needsSync(sourceFile, destFile, opts)

		// Check for conflicts
		if needsSync {
			if conflict := e.resolver.DetectConflict(sourceFile, destFile); conflict != nil {
				atomic.AddInt64(&e.stats.ConflictsFound, 1)

				resolution, err := e.resolver.ResolveConflict(ctx, conflict)
				if err != nil {
					return fmt.Errorf("failed to resolve conflict for %s: %w", sourceFile.Path, err)
				}

				switch resolution {
				case ResolutionSkip:
					return nil
				case ResolutionUseDestination:
					return nil
				case ResolutionBackupAndUseSource:
					if _, err := e.resolver.CreateBackup(destPath); err != nil {
						return fmt.Errorf("failed to create backup: %w", err)
					}

					atomic.AddInt64(&e.stats.ConflictsResolved, 1)
				case ResolutionUseSource:
					atomic.AddInt64(&e.stats.ConflictsResolved, 1)
				}
			}
		}
	}

	if !needsSync {
		return nil
	}

	if opts.DryRun {
		if exists {
			atomic.AddInt64(&e.stats.FilesModified, 1)
		} else {
			atomic.AddInt64(&e.stats.FilesCreated, 1)
		}

		return nil
	}

	if sourceFile.IsDir {
		if err := os.MkdirAll(destPath, os.FileMode(sourceFile.Mode)); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", destPath, err)
		}

		atomic.AddInt64(&e.stats.FilesCreated, 1)

		return nil
	}

	copyErr := e.retryManager.ExecuteWithRetry(ctx, func() error {
		return e.copier.CopyFile(ctx, sourceFile.Path, destPath)
	})
	if copyErr != nil {
		syncErr := ClassifySyncError("copy", sourceFile.Path, copyErr)
		e.errorHandler.AddError(syncErr)
		atomic.AddInt64(&e.stats.ErrorsEncountered, 1)

		return fmt.Errorf("failed to copy file %s to %s after retries: %w", sourceFile.Path, destPath, copyErr)
	}

	atomic.AddInt64(&e.stats.BytesTransferred, sourceFile.Size)

	if exists {
		atomic.AddInt64(&e.stats.FilesModified, 1)
	} else {
		atomic.AddInt64(&e.stats.FilesCreated, 1)
	}

	atomic.AddInt64(&e.stats.FilesChanged, 1)

	return nil
}

func (e *SyncEngine) needsSync(source, dest *FileInfo, opts SyncOptions) bool {
	if source.Size != dest.Size {
		return true
	}

	if !source.ModTime.Equal(dest.ModTime) {
		return true
	}

	if opts.ChecksumVerify && source.Checksum != "" && dest.Checksum != "" {
		return source.Checksum != dest.Checksum
	}

	return false
}

// Watch monitors file changes and automatically synchronizes based on configuration.
func (e *SyncEngine) Watch(ctx context.Context, configPath string) error {
	loader := config.NewLoader()

	cfg, err := loader.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	profile := cfg.Default
	if profile == nil {
		return fmt.Errorf("no default profile found in config")
	}

	if profile.Source == "" || profile.Destination == "" {
		return fmt.Errorf("source and destination must be specified for watch mode")
	}

	if err := e.watcher.Add(profile.Source); err != nil {
		return fmt.Errorf("failed to watch source directory: %w", err)
	}

	if err := e.watcher.Start(ctx); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	go e.handleWatchEvents(ctx, profile)

	<-ctx.Done()

	return e.watcher.Stop()
}

func (e *SyncEngine) handleWatchEvents(ctx context.Context, profile *config.Profile) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-e.watcher.Events():
			e.handleChangeEvent(ctx, event, profile)
		case err := <-e.watcher.Errors():
			fmt.Printf("Watcher error: %v\n", err)
		}
	}
}

func (e *SyncEngine) handleChangeEvent(ctx context.Context, event ChangeEvent, profile *config.Profile) {
	relPath, err := filepath.Rel(profile.Source, event.Path)
	if err != nil {
		return
	}

	destPath := filepath.Join(profile.Destination, relPath)

	switch event.Type {
	case ChangeCreate, ChangeModify:
		if event.Info != nil && !event.Info.IsDir {
			if err := e.copier.CopyFile(ctx, event.Path, destPath); err != nil {
				fmt.Printf("Failed to sync file %s: %v\n", event.Path, err)
			}
		}
	case ChangeDelete:
		if err := os.Remove(destPath); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Failed to delete file %s: %v\n", destPath, err)
		}
	}
}

// GetStats returns a snapshot of the current synchronization statistics.
func (e *SyncEngine) GetStats() *SyncStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	statsCopy := *e.stats

	return &statsCopy
}

// GetProgress returns a snapshot of current progress information.
func (e *SyncEngine) GetProgress() *Progress {
	e.mu.RLock()
	defer e.mu.RUnlock()

	progressCopy := *e.progress

	return &progressCopy
}

// GetErrors returns the collected synchronization errors.
func (e *SyncEngine) GetErrors() []*SyncError {
	return e.errorHandler.GetErrors()
}

// GetErrorSummary returns a summary count of errors by category.
func (e *SyncEngine) GetErrorSummary() map[ErrorCategory]int {
	return e.errorHandler.GetSummary()
}

// ClearErrors clears all accumulated synchronization errors.
func (e *SyncEngine) ClearErrors() {
	e.errorHandler.Clear()
}

func (e *SyncEngine) resetStats() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.stats = &SyncStats{}
	e.progress = &Progress{}
}

func (e *SyncEngine) updateProgress(currentFile string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.progress.CurrentFile = currentFile
	if e.progress.Total > 0 {
		e.progress.Percentage = float64(e.progress.Current) / float64(e.progress.Total) * 100
	}

	elapsed := time.Since(e.stats.StartTime)
	if elapsed > 0 && e.progress.Current > 0 {
		e.progress.Speed = int64(float64(e.stats.BytesTransferred) / elapsed.Seconds())

		if e.progress.Speed > 0 {
			remaining := e.progress.Total - e.progress.Current
			e.progress.ETA = time.Duration(float64(remaining) / float64(e.progress.Speed) * float64(time.Second))
		}
	}
}
