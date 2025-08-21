#!/bin/bash
set -e

# Cross-compilation script for relay
VERSION=${VERSION:-dev}
BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS="-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.commit=${COMMIT}"

# Platforms to build for
PLATFORMS=(
    "linux/amd64"
    "linux/arm64" 
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
    "freebsd/amd64"
)

echo "Cross-compiling relay for multiple platforms..."
echo "Version: $VERSION"
echo "Build time: $BUILD_TIME"
echo "Commit: $COMMIT"
echo

# Create dist directory
mkdir -p dist

for platform in "${PLATFORMS[@]}"; do
    IFS="/" read -r GOOS GOARCH <<< "$platform"
    
    output_name="relay-${VERSION}-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    echo "Building for $GOOS/$GOARCH..."
    
    env GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 \
        go build -ldflags="$LDFLAGS" -o "dist/$output_name" ./src/cmd/relay
    
    # Create checksum
    if command -v sha256sum >/dev/null; then
        (cd dist && sha256sum "$output_name" > "$output_name.sha256")
    elif command -v shasum >/dev/null; then
        (cd dist && shasum -a 256 "$output_name" > "$output_name.sha256")
    fi
    
    echo "✓ Built dist/$output_name"
done

echo
echo "Cross-compilation complete!"
echo "Binaries available in dist/ directory"