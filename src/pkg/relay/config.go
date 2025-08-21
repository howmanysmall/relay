// Package relay provides public API types and functions for the relay file synchronization tool.
package relay

import (
	"github.com/howmanysmall/relay/src/internal/config"
)

// Config and related types are re-exported for public API access.
type (
	Config = config.Config
	// Profile re-exports config.Profile for public API consumers.
	Profile = config.Profile
	// FilterRules re-exports config.FilterRules for public API consumers.
	FilterRules = config.FilterRules
	// ConflictConfig re-exports config.ConflictConfig for public API consumers.
	ConflictConfig = config.ConflictConfig
	// RetryConfig re-exports config.RetryConfig for public API consumers.
	RetryConfig = config.RetryConfig
	// PerformanceConfig re-exports config.PerformanceConfig for public API consumers.
	PerformanceConfig = config.PerformanceConfig
	// ConflictStrategy re-exports config.ConflictStrategy.
	ConflictStrategy = config.ConflictStrategy
	// SyncMode re-exports config.SyncMode.
	SyncMode = config.SyncMode
	// BackoffStrategy re-exports config.BackoffStrategy.
	BackoffStrategy = config.BackoffStrategy
)

// Re-export constants
const (
	ConflictNewest      = config.ConflictNewest
	ConflictSource      = config.ConflictSource
	ConflictDestination = config.ConflictDestination
	ConflictInteractive = config.ConflictInteractive
	ConflictSmart       = config.ConflictSmart
	ConflictSkip        = config.ConflictSkip

	ModeMirror = config.ModeMirror
	ModeSync   = config.ModeSync
	ModeWatch  = config.ModeWatch

	BackoffLinear      = config.BackoffLinear
	BackoffExponential = config.BackoffExponential
	BackoffFixed       = config.BackoffFixed
)

// LoadConfig loads and validates a configuration file from the specified path.
func LoadConfig(configPath string) (*Config, error) {
	loader := config.NewLoader()

	cfg, err := loader.Load(configPath)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
