// Package config provides configuration loading and validation for the relay CLI tool.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/tidwall/gjson"
)

// Loader handles loading and parsing configuration files from multiple formats.
type Loader struct {
	searchPaths []string
}

// NewLoader creates a new configuration loader with default search paths.
func NewLoader() *Loader {
	return &Loader{
		searchPaths: []string{
			".",
			"~/.config/relay",
			"~/.relay",
		},
	}
}

// Load loads configuration from the specified path or searches for default config files.
func (l *Loader) Load(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = l.findDefaultConfig()
	}

	if configPath == "" {
		return l.getDefaultConfig(), nil
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	ext := strings.ToLower(filepath.Ext(configPath))

	config, err := l.parseByExtension(content, ext)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	if err := l.validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if err := l.resolveExtends(config); err != nil {
		return nil, fmt.Errorf("failed to resolve profile inheritance: %w", err)
	}

	return config, nil
}

func (l *Loader) findDefaultConfig() string {
	candidates := []string{
		"relay.jsonc",
		"relay.json",
		"relay.toml",
		".relay.jsonc",
		".relay.json",
		".relay.toml",
	}

	for _, searchPath := range l.searchPaths {
		for _, candidate := range candidates {
			fullPath := filepath.Join(searchPath, candidate)
			if _, err := os.Stat(fullPath); err == nil {
				return fullPath
			}
		}
	}

	return ""
}

func (l *Loader) parseByExtension(content []byte, ext string) (*Config, error) {
	var config Config

	switch ext {
	case ".json", ".jsonc":
		// Handle JSONC by stripping comments
		cleaned := l.stripJSONComments(string(content))
		if err := json.Unmarshal([]byte(cleaned), &config); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
	case ".toml":
		if err := toml.Unmarshal(content, &config); err != nil {
			return nil, fmt.Errorf("invalid TOML: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config format: %s", ext)
	}

	return &config, nil
}

func (l *Loader) stripJSONComments(content string) string {
	result := gjson.Parse(content)
	if !result.IsObject() {
		return content
	}

	var cleaned map[string]interface{}
	if err := json.Unmarshal([]byte(content), &cleaned); err != nil {
		// If standard JSON parsing fails, try manual comment removal
		lines := strings.Split(content, "\n")

		var cleanedLines []string

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}

			if idx := strings.Index(line, "//"); idx != -1 {
				line = line[:idx]
			}

			cleanedLines = append(cleanedLines, line)
		}

		return strings.Join(cleanedLines, "\n")
	}

	cleanedJSON, _ := json.Marshal(cleaned)

	return string(cleanedJSON)
}

func (l *Loader) validateConfig(config *Config) error {
	if config.Version == "" {
		config.Version = "1.0"
	}

	if config.Default == nil && len(config.Profiles) == 0 {
		return fmt.Errorf("config must have either a default profile or named profiles")
	}

	// Validate each profile
	if config.Default != nil {
		if err := l.validateProfile(config.Default); err != nil {
			return fmt.Errorf("invalid default profile: %w", err)
		}
	}

	for name, profile := range config.Profiles {
		if err := l.validateProfile(profile); err != nil {
			return fmt.Errorf("invalid profile %s: %w", name, err)
		}
	}

	return nil
}

func (l *Loader) validateProfile(profile *Profile) error {
	if profile.Mode == "" {
		profile.Mode = string(ModeMirror)
	}

	validModes := []string{string(ModeMirror), string(ModeSync), string(ModeWatch)}
	isValidMode := false

	for _, mode := range validModes {
		if profile.Mode == mode {
			isValidMode = true
			break
		}
	}

	if !isValidMode {
		return fmt.Errorf("invalid mode %s, must be one of: %v", profile.Mode, validModes)
	}

	if profile.Workers < 0 {
		return fmt.Errorf("workers must be non-negative, got %d", profile.Workers)
	}

	// Set defaults
	if profile.Workers == 0 {
		profile.Workers = -1 // Auto-detect
	}

	if profile.BufferSize == "" {
		profile.BufferSize = "auto"
	}

	if profile.Conflict != nil {
		if err := l.validateConflictConfig(profile.Conflict); err != nil {
			return fmt.Errorf("invalid conflict config: %w", err)
		}
	}

	if profile.Retry != nil {
		l.validateRetryConfig(profile.Retry)
	}

	return nil
}

func (l *Loader) validateConflictConfig(config *ConflictConfig) error {
	if config.Strategy == "" {
		config.Strategy = string(ConflictNewest)
	}

	validStrategies := []string{
		string(ConflictNewest),
		string(ConflictSource),
		string(ConflictDestination),
		string(ConflictInteractive),
		string(ConflictSmart),
		string(ConflictSkip),
	}

	isValid := false

	for _, strategy := range validStrategies {
		if config.Strategy == strategy {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("invalid conflict strategy %s, must be one of: %v", config.Strategy, validStrategies)
	}

	if config.BackupDir == "" && config.Backup {
		config.BackupDir = ".relay-backups"
	}

	return nil
}

func (l *Loader) validateRetryConfig(config *RetryConfig) {
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 3
	}

	if config.InitialDelay == 0 {
		config.InitialDelay = 100 * time.Millisecond
	}

	if config.MaxDelay == 0 {
		config.MaxDelay = 10 * time.Second
	}

	if config.Multiplier <= 0 {
		config.Multiplier = 2.0
	}

	if config.Backoff == "" {
		config.Backoff = string(BackoffExponential)
	}
}

func (l *Loader) resolveExtends(config *Config) error {
	// Resolve inheritance for named profiles
	for name, profile := range config.Profiles {
		if profile.Extends != "" {
			if err := l.applyExtends(profile, config); err != nil {
				return fmt.Errorf("failed to resolve extends for profile %s: %w", name, err)
			}
		}
	}

	return nil
}

func (l *Loader) applyExtends(profile *Profile, config *Config) error {
	if profile.Extends == "default" && config.Default != nil {
		l.mergeProfiles(profile, config.Default)
	} else if baseProfile, exists := config.Profiles[profile.Extends]; exists {
		l.mergeProfiles(profile, baseProfile)
	} else {
		return fmt.Errorf("extended profile %s not found", profile.Extends)
	}

	return nil
}

func (l *Loader) mergeProfiles(target, base *Profile) {
	if target.Mode == "" {
		target.Mode = base.Mode
	}

	if target.Source == "" {
		target.Source = base.Source
	}

	if target.Destination == "" {
		target.Destination = base.Destination
	}

	if target.Workers == 0 {
		target.Workers = base.Workers
	}

	if target.BufferSize == "" {
		target.BufferSize = base.BufferSize
	}

	if target.Filters == nil {
		target.Filters = base.Filters
	}

	if target.Conflict == nil {
		target.Conflict = base.Conflict
	}

	if target.Retry == nil {
		target.Retry = base.Retry
	}

	if target.Performance == nil {
		target.Performance = base.Performance
	}
}

func (l *Loader) getDefaultConfig() *Config {
	return &Config{
		Version: "1.0",
		Default: &Profile{
			Mode:       string(ModeMirror),
			Watch:      false,
			Workers:    -1, // Auto-detect
			BufferSize: "auto",
			Filters: &FilterRules{
				Smart:            true,
				Include:          []string{"**/*"},
				Exclude:          []string{".git", "node_modules", "*.tmp", "*.swp"},
				RespectGitignore: true,
				IgnoreHidden:     false,
			},
			Conflict: &ConflictConfig{
				Strategy:    string(ConflictNewest),
				Backup:      false,
				Interactive: false,
			},
			Retry: &RetryConfig{
				MaxAttempts:  3,
				InitialDelay: 100 * time.Millisecond,
				MaxDelay:     10 * time.Second,
				Multiplier:   2.0,
				Backoff:      string(BackoffExponential),
			},
			Performance: &PerformanceConfig{
				UseZeroCopy:   true,
				EnableCaching: true,
				ChecksumAlgo:  "blake3",
				IOConcurrency: -1, // Auto-detect
			},
		},
	}
}
