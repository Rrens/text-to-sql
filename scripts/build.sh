#!/bin/bash
# scripts/build.sh - Build script for multiple platforms

set -e

VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS="-s -w -X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME"

echo "Building Text-to-SQL v$VERSION"

# Clean
rm -rf bin/
mkdir -p bin/

# Build for current platform
echo "Building for current platform..."
go build -ldflags "$LDFLAGS" -o bin/server ./cmd/server
echo "✓ Built bin/server"

# Build for Linux AMD64
echo "Building for Linux AMD64..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o bin/server-linux-amd64 ./cmd/server
echo "✓ Built bin/server-linux-amd64"

# Build for Linux ARM64
echo "Building for Linux ARM64..."
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o bin/server-linux-arm64 ./cmd/server
echo "✓ Built bin/server-linux-arm64"

# Build for macOS AMD64
echo "Building for macOS AMD64..."
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o bin/server-darwin-amd64 ./cmd/server
echo "✓ Built bin/server-darwin-amd64"

# Build for macOS ARM64
echo "Building for macOS ARM64..."
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o bin/server-darwin-arm64 ./cmd/server
echo "✓ Built bin/server-darwin-arm64"

# Calculate checksums
echo ""
echo "Calculating checksums..."
cd bin
sha256sum server-* > checksums.txt 2>/dev/null || shasum -a 256 server-* > checksums.txt
cat checksums.txt
cd ..

echo ""
echo "Build complete!"
ls -lah bin/
