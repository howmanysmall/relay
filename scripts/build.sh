#!/bin/bash
set -e

# Build script for relay
VERSION=${VERSION:-dev}
BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS="-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.commit=${COMMIT}"

echo "Building relay..."
echo "Version: $VERSION"
echo "Build time: $BUILD_TIME"
echo "Commit: $COMMIT"

# Create bin directory
mkdir -p bin

# Build for current platform
go build -ldflags="$LDFLAGS" -o bin/relay ./src/cmd/relay

echo "Build complete: bin/relay"