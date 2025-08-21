package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewLoader(t *testing.T) {
	t.Parallel()

	loader := NewLoader()
	if loader == nil {
		t.Fatal("NewLoader returned nil")
	}
}

func TestLoaderLoadJSON(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.json")

	jsonContent := `{
		"profiles": {
			"default": {
				"source": "/path/to/source",
				"destination": "/path/to/dest",
				"mode": "mirror",
				"filters": {
					"include": ["*.txt", "*.md"],
					"exclude": ["*.tmp"]
				},
				"conflict": {
					"strategy": "newest",
					"backup": false
				},
				"retry": {
					"maxAttempts": 3,
					"backoffStrategy": "exponential",
					"baseDelayMs": 1000
				},
				"performance": {
					"ioConcurrency": 4,
					"bufferSizeKb": 64,
					"checksumAlgo": "blake3"
				}
			}
		}
	}`

	if err := os.WriteFile(configFile, []byte(jsonContent), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	loader := NewLoader()

	config, err := loader.Load(configFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify config structure
	if config.Profiles == nil {
		t.Fatal("Profiles is nil")
	}

	defaultProfile, exists := config.Profiles["default"]
	if !exists {
		t.Fatal("Default profile not found")
	}

	// Verify profile fields
	if defaultProfile.Source != "/path/to/source" {
		t.Errorf("Source = %v, want /path/to/source", defaultProfile.Source)
	}

	if defaultProfile.Destination != "/path/to/dest" {
		t.Errorf("Destination = %v, want /path/to/dest", defaultProfile.Destination)
	}

	if defaultProfile.Mode != "mirror" {
		t.Errorf("Mode = %v, want mirror", defaultProfile.Mode)
	}

	// Verify filters
	if len(defaultProfile.Filters.Include) != 2 {
		t.Errorf("Include filters count = %d, want 2", len(defaultProfile.Filters.Include))
	}

	if len(defaultProfile.Filters.Exclude) != 1 {
		t.Errorf("Exclude filters count = %d, want 1", len(defaultProfile.Filters.Exclude))
	}

	// Verify conflict config
	if defaultProfile.Conflict.Strategy != "newest" {
		t.Errorf("Conflict strategy = %v, want newest", defaultProfile.Conflict.Strategy)
	}

	// Verify retry config
	if defaultProfile.Retry.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", defaultProfile.Retry.MaxAttempts)
	}

	if defaultProfile.Retry.Backoff != "exponential" {
		t.Errorf("Backoff = %v, want exponential", defaultProfile.Retry.Backoff)
	}

	// Verify performance config
	if defaultProfile.Performance.IOConcurrency != 4 {
		t.Errorf("IOConcurrency = %d, want 4", defaultProfile.Performance.IOConcurrency)
	}

	if defaultProfile.Performance.ChecksumAlgo != "blake3" {
		t.Errorf("ChecksumAlgo = %v, want blake3", defaultProfile.Performance.ChecksumAlgo)
	}
}

func TestLoaderLoadJSONC(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.jsonc")

	jsoncContent := `{
		// Main configuration
		"profiles": {
			// Default sync profile
			"default": {
				"source": "/path/to/source",
				"destination": "/path/to/dest",
				"mode": "sync", // Two-way sync
				"filters": {
					"include": ["*.go", "*.md"],
					"exclude": ["*.tmp", "*.log"] // Temporary files
				}
			}
		}
	}`

	if err := os.WriteFile(configFile, []byte(jsoncContent), 0o644); err != nil {
		t.Fatalf("Failed to write JSONC config file: %v", err)
	}

	loader := NewLoader()

	config, err := loader.Load(configFile)
	if err != nil {
		t.Fatalf("Load JSONC failed: %v", err)
	}

	defaultProfile := config.Profiles["default"]
	if defaultProfile.Mode != "sync" {
		t.Errorf("Mode = %v, want sync", defaultProfile.Mode)
	}

	if len(defaultProfile.Filters.Include) != 2 {
		t.Errorf("Include filters count = %d, want 2", len(defaultProfile.Filters.Include))
	}
}

func TestLoaderLoadTOML(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.toml")

	tomlContent := `
[profiles.default]
source = "/path/to/source"
destination = "/path/to/dest"
mode = "watch"

[profiles.default.filters]
include = ["*.txt"]
exclude = ["*.tmp"]

[profiles.default.conflict]
strategy = "source"
backup = true

[profiles.default.performance]
ioConcurrency = 8
bufferSizeKb = 128
checksumAlgo = "sha256"
`

	if err := os.WriteFile(configFile, []byte(tomlContent), 0o644); err != nil {
		t.Fatalf("Failed to write TOML config file: %v", err)
	}

	loader := NewLoader()

	config, err := loader.Load(configFile)
	if err != nil {
		t.Fatalf("Load TOML failed: %v", err)
	}

	defaultProfile := config.Profiles["default"]
	if defaultProfile.Mode != "watch" {
		t.Errorf("Mode = %v, want watch", defaultProfile.Mode)
	}

	if defaultProfile.Conflict.Strategy != "source" {
		t.Errorf("Conflict strategy = %v, want source", defaultProfile.Conflict.Strategy)
	}

	if !defaultProfile.Conflict.Backup {
		t.Errorf("Backup should be true")
	}

	if defaultProfile.Performance.IOConcurrency != 8 {
		t.Errorf("IOConcurrency = %d, want 8", defaultProfile.Performance.IOConcurrency)
	}

	if defaultProfile.Performance.ChecksumAlgo != "sha256" {
		t.Errorf("ChecksumAlgo = %v, want sha256", defaultProfile.Performance.ChecksumAlgo)
	}
}

func TestLoaderProfileInheritance(t *testing.T) {
	t.Skip("Profile inheritance not implemented yet")
	t.Parallel()
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_inheritance.json")

	jsonContent := `{
		"profiles": {
			"base": {
				"mode": "mirror",
				"filters": {
					"include": ["*.txt"]
				},
				"performance": {
					"ioConcurrency": 4,
					"checksumAlgo": "blake3"
				}
			},
			"production": {
				"extends": "base",
				"source": "/prod/source",
				"destination": "/prod/dest",
				"performance": {
					"ioConcurrency": 8
				}
			}
		}
	}`

	if err := os.WriteFile(configFile, []byte(jsonContent), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	loader := NewLoader()

	config, err := loader.Load(configFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	prodProfile := config.Profiles["production"]

	// Should inherit mode from base
	if prodProfile.Mode != "mirror" {
		t.Errorf("Mode = %v, want mirror (inherited)", prodProfile.Mode)
	}

	// Should inherit filters from base
	if len(prodProfile.Filters.Include) != 1 || prodProfile.Filters.Include[0] != "*.txt" {
		t.Errorf("Include filters not inherited correctly")
	}

	// Should have its own source/destination
	if prodProfile.Source != "/prod/source" {
		t.Errorf("Source = %v, want /prod/source", prodProfile.Source)
	}

	// Should override IOConcurrency but inherit ChecksumAlgo
	if prodProfile.Performance.IOConcurrency != 8 {
		t.Errorf("IOConcurrency = %d, want 8 (overridden)", prodProfile.Performance.IOConcurrency)
	}

	if prodProfile.Performance.ChecksumAlgo != "blake3" {
		t.Errorf("ChecksumAlgo = %v, want blake3 (inherited)", prodProfile.Performance.ChecksumAlgo)
	}
}

func TestLoaderNonExistentFile(t *testing.T) {
	t.Parallel()

	loader := NewLoader()

	_, err := loader.Load("/nonexistent/config.json")
	if err == nil {
		t.Errorf("Expected error when loading nonexistent file")
	}
}

func TestLoaderInvalidJSON(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "invalid.json")

	invalidJSON := `{
		"profiles": {
			"default": {
				"source": "/path"
				// Missing comma - invalid JSON
				"destination": "/dest"
			}
		}
	}`

	if err := os.WriteFile(configFile, []byte(invalidJSON), 0o644); err != nil {
		t.Fatalf("Failed to write invalid JSON file: %v", err)
	}

	loader := NewLoader()

	_, err := loader.Load(configFile)
	if err == nil {
		t.Errorf("Expected error when loading invalid JSON")
	}
}

func TestLoaderAutoDetectFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		filename   string
		content    string
		wantSource string
	}{
		{
			name:       "JSON format",
			filename:   "config.json",
			content:    `{"profiles": {"default": {"source": "/json/path"}}}`,
			wantSource: "/json/path",
		},
		{
			name:       "JSONC format",
			filename:   "config.jsonc",
			content:    `{"profiles": {"default": {"source": "/jsonc/path"}}}`,
			wantSource: "/jsonc/path",
		},
		{
			name:       "TOML format",
			filename:   "config.toml",
			content:    "[profiles.default]\nsource = \"/toml/path\"",
			wantSource: "/toml/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tempDir := t.TempDir()
			configFile := filepath.Join(tempDir, tt.filename)

			if err := os.WriteFile(configFile, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			loader := NewLoader()

			config, err := loader.Load(configFile)
			if err != nil {
				t.Fatalf("Load failed: %v", err)
			}

			if config.Profiles["default"].Source != tt.wantSource {
				t.Errorf("Source = %v, want %v", config.Profiles["default"].Source, tt.wantSource)
			}
		})
	}
}
