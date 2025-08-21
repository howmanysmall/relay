# Relay

High-performance file mirroring and synchronization tool built for speed,
usability, and reliability.

## Features

- **Blazing Fast**: Optimized for maximum performance with zero-copy operations
and parallel processing
- **One-Way Mirroring**: Efficiently mirror files from source to destination
- **Two-Way Sync**: Intelligent bidirectional synchronization with conflict resolution
- **Real-Time Watching**: Monitor directories for changes and sync automatically
- **Beautiful UI**: Interactive dashboard and colored terminal output
- **Smart Filtering**: Auto-exclude build artifacts, respect .gitignore, custom patterns
- **Cross-Platform**: Works on Linux, macOS, and Windows
- **Configuration**: Flexible JSON/JSONC/TOML configuration files

## Installation

### Build from Source

```bash
# Clone the repository
git clone https://github.com/howmanysmall/relay.git
cd relay

# Build using the provided script
./scripts/build.sh

# The binary will be available at bin/relay
```

### Manual Build

```bash
go build -o bin/relay ./src/cmd/relay
```

## Quick Start

### Basic File Mirroring

```bash
# Mirror source directory to destination (one-way)
relay mirror ./source ./backup

# Preview changes without copying
relay mirror ./source ./backup --dry-run

# Verbose output
relay mirror ./source ./backup --verbose
```

### Two-Way Synchronization

```bash
# Synchronize two directories
relay sync ./local ./remote

# Prefer local changes in conflicts
relay sync ./local ./remote --prefer-local

# Interactive conflict resolution
relay sync ./local ./remote --ask
```

### Real-Time Watching

```bash
# Watch directories using default config
relay watch

# Watch with custom configuration
relay watch --config myproject.jsonc

# Watch with live dashboard
relay watch --dashboard
```

## Commands

### `relay mirror <source> <destination>`

One-way file mirroring from source to destination.

**Examples:**

```bash
# Basic mirror
relay mirror ./source ./backup

# Only copy newer files
relay mirror ./src ./dst --if-newer

# Auto-exclude build artifacts
relay mirror ./project ./backup --smart

# Maximum performance mode
relay mirror ./src ./dst --turbo

# Low resource usage
relay mirror ./docs ./web --gentle

# Only changes from last hour
relay mirror ./src ./dst --since 1h

# Custom include/exclude patterns
relay mirror ./src ./dst --include "*.go" --exclude "*.tmp"
```

### `relay sync <path1> <path2>`

Two-way synchronization between directories.

**Examples:**

```bash
# Basic two-way sync
relay sync ./local ./remote

# Local changes win conflicts
relay sync ./a ./b --prefer-local

# Remote changes win conflicts
relay sync ./a ./b --prefer-remote

# Interactive conflict resolution
relay sync ./a ./b --ask

# Create backups before overwriting
relay sync ./docs ./backup --backup
```

### `relay watch`

Watch directories for changes and sync in real-time.

**Examples:**

```bash
# Use default configuration (relay.jsonc)
relay watch

# Specify custom config file
relay watch --config production.jsonc

# Show live dashboard
relay watch --dashboard

# Set minimum sync interval
relay watch --interval 500ms

# Preview mode
relay watch --dry-run
```

### `relay validate <config-file>`

Validate configuration files.

**Examples:**

```bash
# Validate default config
relay validate

# Validate specific config
relay validate configs/production.jsonc

# Validate with verbose output
relay validate myconfig.toml --verbose
```

## Global Options

All commands support these global flags:

```bash
--config string      Config file (default: relay.jsonc)
--verbose, -v        Verbose output
--dry-run           Preview changes without executing
--workers int       Number of worker goroutines (0 = auto)
--buffer string     Buffer size for operations (default: auto)
--profile string    Configuration profile to use (default: default)
```

## Configuration

Relay supports JSON, JSONC (with comments), and TOML configuration files.

### Basic Configuration

Create a `relay.jsonc` file:

```jsonc
{
	"default": {
		"source": "./source",
		"destination": "./destination",
		"mode": "mirror",
		"watch": false,
		"filters": {
			"exclude": [".git", "node_modules", "*.tmp"],
			"smart": true
		}
	}
}
```

### Advanced Configuration

```jsonc
{
	"default": {
		"bufferSize": "10MB",
		"conflict": {
			"backup": true,
			"backupDir": ".relay-backups",
			"strategy": "newest"
		},

		"destination": "./backup",
		"filters": {
			"exclude": [".git", "node_modules", "target", "*.log"],
			"include": ["**/*"],
			"maxFileSize": "1GB",
			"respectGitignore": true,
			"smart": true
		},

		"mode": "mirror",
		"performance": {
			"checksumAlgo": "blake3",
			"enableCaching": true,
			"ioConcurrency": 16,
			"useZeroCopy": true
		},

		"retry": {
			"backoff": "exponential",
			"initialDelay": "100ms",
			"maxAttempts": 3
		},

		"source": "./src",
		"workers": 8
	}
}

```

### Multiple Profiles

```bash
# Use specific profile
relay mirror ./src ./dst --profile production

# Configuration with multiple profiles
{
	"profiles": {
		"development": {
			"extends": "default",
			"filters": {
				"exclude": [".git", "node_modules", "target"]
			},
			"watch": true
		},
		"production": {
			"extends": "default",
			"performance": {
				"ioConcurrency": 32,
				"useZeroCopy": true
			},
			"workers": 16
		}
	}
}

```

## Common Use Cases

### Development Workflow

```bash
# Mirror source to build directory, excluding dev files
relay mirror ./src ./build --smart --if-newer

# Two-way sync between local and remote development
relay sync ./local-dev ./remote-dev --prefer-local

# Watch for changes during development
relay watch --config dev.jsonc --dashboard
```

### Backup and Archival

```bash
# Create backup with compression-friendly settings
relay mirror ./important ./backup --smart

# Incremental backup (only newer files)
relay mirror ./documents ./backup --if-newer

# Backup with custom exclusions
relay mirror ./home ./backup --exclude "*.cache" --exclude "*.tmp"
```

### Deployment and CI/CD

```bash
# Deploy built assets
relay mirror ./dist ./production --turbo

# Sync configuration files
relay sync ./configs ./deployed-configs --prefer-local --backup

# Fast deployment with maximum performance
relay mirror ./build ./deploy --turbo --workers 16 --buffer 50MB
```

## Performance Tuning

### High-Performance Mode

```bash
# Maximum speed for large transfers
relay mirror ./source ./dest --turbo --workers 16 --buffer 50MB

# Zero-copy optimization (when supported)
relay mirror ./source ./dest --turbo
```

### Low-Resource Mode

```bash
# Gentle mode for background operations
relay mirror ./source ./dest --gentle --workers 2

# Limit buffer size
relay mirror ./source ./dest --buffer 1MB
```

### Custom Worker Configuration

```bash
# Specific worker count
relay mirror ./src ./dst --workers 8

# Auto-detect optimal workers
relay mirror ./src ./dst --workers 0
```

## Filtering Examples

### Include/Exclude Patterns

```bash
# Only copy Go files
relay mirror ./src ./dst --include "*.go"

# Exclude temporary files
relay mirror ./src ./dst --exclude "*.tmp" --exclude "*.swp"

# Multiple patterns
relay mirror ./src ./dst --include "*.go" --include "*.md" --exclude "test_*"
```

### Smart Filtering

```bash
# Auto-exclude common build artifacts
relay mirror ./project ./backup --smart

# Respect .gitignore patterns
relay mirror ./repo ./backup --smart
```

### Time-Based Filtering

```bash
# Only files changed in last hour
relay mirror ./src ./dst --since 1h

# Files changed in last 2 days
relay mirror ./src ./dst --since 2d

# Files changed since specific time
relay mirror ./src ./dst --since 2024-01-01
```

## Building from Source

### Prerequisites

- Go 1.25.0 or later
- Git

### Build Process

```bash
# Clone repository
git clone https://github.com/howmanysmall/relay.git
cd relay

# Build for current platform
./scripts/build.sh

# Cross-compile for multiple platforms
./scripts/cross-build.sh

# Manual build with custom flags
go build -ldflags="-s -w" -o bin/relay ./src/cmd/relay
```

### Development

```bash
# Run tests
go test ./...

# Run linter
golangci-lint run

# Format code
go fmt ./...
```

## Project Status

### Implemented Features ✅

- ✅ One-way mirroring (`relay mirror`)
- ✅ Configuration file support (JSON/JSONC/TOML)
- ✅ File filtering and smart exclusions
- ✅ Performance optimizations
- ✅ Interactive dashboard UI
- ✅ Dry-run mode
- ✅ Configuration validation

### Planned Features 🚧

- 🚧 Two-way synchronization (`relay sync`)
- 🚧 Real-time watching (`relay watch`)
- 🚧 Conflict resolution
- 🚧 Network synchronization
- 🚧 Incremental sync optimization

## License

MIT License

Copyright (c) 2025 howmanysmall

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

## Contributing

Just follow the contribution guide. 😃
