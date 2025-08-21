package core

import (
	"context"
	"time"
)

// FileInfo contains metadata about a file or directory.
type FileInfo struct {
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"modTime"`
	Mode         uint32    `json:"mode"`
	IsDir        bool      `json:"isDir"`
	Checksum     string    `json:"checksum,omitempty"`
	ChecksumAlgo string    `json:"checksumAlgo,omitempty"`
}

// ChangeEvent represents a file system change event.
type ChangeEvent struct {
	Type      ChangeType `json:"type"`
	Path      string     `json:"path"`
	OldPath   string     `json:"oldPath,omitempty"`
	Info      *FileInfo  `json:"info,omitempty"`
	Timestamp time.Time  `json:"timestamp"`
}

// ChangeType represents the type of file system change.
type ChangeType int

// File system change types
const (
	ChangeCreate ChangeType = iota
	ChangeModify
	ChangeDelete
	ChangeRename
)

func (ct ChangeType) String() string {
	switch ct {
	case ChangeCreate:
		return "create"
	case ChangeModify:
		return "modify"
	case ChangeDelete:
		return "delete"
	case ChangeRename:
		return "rename"
	default:
		return "unknown"
	}
}

// SyncStats contains statistics about a synchronization operation.
type SyncStats struct {
	FilesScanned      int64         `json:"filesScanned"`
	FilesChanged      int64         `json:"filesChanged"`
	FilesCreated      int64         `json:"filesCreated"`
	FilesModified     int64         `json:"filesModified"`
	FilesDeleted      int64         `json:"filesDeleted"`
	BytesTransferred  int64         `json:"bytesTransferred"`
	ConflictsFound    int64         `json:"conflictsFound"`
	ConflictsResolved int64         `json:"conflictsResolved"`
	ErrorsEncountered int64         `json:"errorsEncountered"`
	StartTime         time.Time     `json:"startTime"`
	EndTime           time.Time     `json:"endTime,omitempty"`
	Duration          time.Duration `json:"duration"`
}

// Progress tracks the progress of a synchronization operation.
type Progress struct {
	Current     int64         `json:"current"`
	Total       int64         `json:"total"`
	Percentage  float64       `json:"percentage"`
	Speed       int64         `json:"speed"`
	ETA         time.Duration `json:"eta"`
	CurrentFile string        `json:"currentFile"`
}

// SyncOptions configures synchronization behavior.
type SyncOptions struct {
	DryRun           bool          `json:"dryRun"`
	Force            bool          `json:"force"`
	Recursive        bool          `json:"recursive"`
	PreservePerms    bool          `json:"preservePerms"`
	PreserveTimes    bool          `json:"preserveTimes"`
	DeleteExtraneous bool          `json:"deleteExtraneous"`
	ChecksumVerify   bool          `json:"checksumVerify"`
	Workers          int           `json:"workers"`
	BufferSize       int64         `json:"bufferSize"`
	Timeout          time.Duration `json:"timeout"`
}

// Watcher interface for monitoring file system changes.
type Watcher interface {
	Start(ctx context.Context) error
	Stop() error
	Add(path string) error
	Remove(path string) error
	Events() <-chan ChangeEvent
	Errors() <-chan error
}

// Scanner interface for scanning directories.
type Scanner interface {
	Scan(ctx context.Context, path string) ([]*FileInfo, error)
	ScanWithFilter(ctx context.Context, path string, filter FilterFunc) ([]*FileInfo, error)
}

// Syncer interface for synchronizing files.
type Syncer interface {
	Sync(ctx context.Context, source, destination string, opts SyncOptions) (*SyncStats, error)
	SyncWithProgress(ctx context.Context, source, destination string, opts SyncOptions, progressCh chan<- Progress) (*SyncStats, error)
}

// FilterFunc is a function that determines if a file should be processed.
type FilterFunc func(path string, info *FileInfo) bool

// Engine interface for the main synchronization engine.
type Engine interface {
	Mirror(ctx context.Context, source, destination string) error
	Sync(ctx context.Context, path1, path2 string) error
	Watch(ctx context.Context, configPath string) error
	GetStats() *SyncStats
	GetProgress() *Progress
}
