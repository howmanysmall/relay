package config

import (
	"time"
)

// Config represents the main configuration structure for relay.
type Config struct {
	Version  string              `json:"version" toml:"version"`
	Default  *Profile            `json:"default,omitempty" toml:"default,omitempty"`
	Profiles map[string]*Profile `json:"profiles,omitempty" toml:"profiles,omitempty"`
}

// Profile defines synchronization settings and behavior.
type Profile struct {
	Mode        string             `json:"mode" toml:"mode"`
	Source      string             `json:"source,omitempty" toml:"source,omitempty"`
	Destination string             `json:"destination,omitempty" toml:"destination,omitempty"`
	Watch       bool               `json:"watch" toml:"watch"`
	Workers     int                `json:"workers" toml:"workers"`
	BufferSize  string             `json:"bufferSize" toml:"bufferSize"`
	Filters     *FilterRules       `json:"filters,omitempty" toml:"filters,omitempty"`
	Conflict    *ConflictConfig    `json:"conflict,omitempty" toml:"conflict,omitempty"`
	Retry       *RetryConfig       `json:"retry,omitempty" toml:"retry,omitempty"`
	Performance *PerformanceConfig `json:"performance,omitempty" toml:"performance,omitempty"`
	Extends     string             `json:"extends,omitempty" toml:"extends,omitempty"`
}

// FilterRules defines file filtering and exclusion patterns.
type FilterRules struct {
	Smart            bool     `json:"smart" toml:"smart"`
	Include          []string `json:"include" toml:"include"`
	Exclude          []string `json:"exclude" toml:"exclude"`
	RespectGitignore bool     `json:"respectGitignore" toml:"respectGitignore"`
	IgnoreHidden     bool     `json:"ignoreHidden" toml:"ignoreHidden"`
	MaxFileSize      string   `json:"maxFileSize,omitempty" toml:"maxFileSize,omitempty"`
	MinFileSize      string   `json:"minFileSize,omitempty" toml:"minFileSize,omitempty"`
}

// ConflictConfig defines how file conflicts should be resolved.
type ConflictConfig struct {
	Strategy    string `json:"strategy" toml:"strategy"`
	Backup      bool   `json:"backup" toml:"backup"`
	BackupDir   string `json:"backupDir,omitempty" toml:"backupDir,omitempty"`
	Interactive bool   `json:"interactive" toml:"interactive"`
}

// RetryConfig defines retry behavior for failed operations.
type RetryConfig struct {
	MaxAttempts  int           `json:"maxAttempts" toml:"maxAttempts"`
	InitialDelay time.Duration `json:"initialDelay" toml:"initialDelay"`
	MaxDelay     time.Duration `json:"maxDelay" toml:"maxDelay"`
	Multiplier   float64       `json:"multiplier" toml:"multiplier"`
	Backoff      string        `json:"backoff" toml:"backoff"`
}

// PerformanceConfig defines performance optimization settings.
type PerformanceConfig struct {
	UseZeroCopy    bool          `json:"useZeroCopy" toml:"useZeroCopy"`
	EnableCaching  bool          `json:"enableCaching" toml:"enableCaching"`
	ChecksumAlgo   string        `json:"checksumAlgo" toml:"checksumAlgo"`
	IOConcurrency  int           `json:"ioConcurrency" toml:"ioConcurrency"`
	NetworkTimeout time.Duration `json:"networkTimeout" toml:"networkTimeout"`
}

// ConflictStrategy represents different conflict resolution strategies
type ConflictStrategy string

// Conflict resolution strategies
const (
	ConflictNewest      ConflictStrategy = "newest"
	ConflictSource      ConflictStrategy = "source"
	ConflictDestination ConflictStrategy = "destination"
	ConflictInteractive ConflictStrategy = "interactive"
	ConflictSmart       ConflictStrategy = "smart"
	ConflictSkip        ConflictStrategy = "skip"
)

// SyncMode represents different synchronization modes
type SyncMode string

// Synchronization modes
const (
	ModeMirror SyncMode = "mirror"
	ModeSync   SyncMode = "sync"
	ModeWatch  SyncMode = "watch"
)

// BackoffStrategy represents different retry backoff strategies
type BackoffStrategy string

// Retry backoff strategies
const (
	BackoffLinear      BackoffStrategy = "linear"
	BackoffExponential BackoffStrategy = "exponential"
	BackoffFixed       BackoffStrategy = "fixed"
)
