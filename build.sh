#!/bin/bash

VERSION="dev"

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
. "$SCRIPT_DIR/scripts/version.sh"

echo "Building version: $VERSION"

export CGO_ENABLED=0

echo "Downloading dependencies..."
go mod tidy

WIN_OUT="sfsd_win_amd64_${VERSION}.exe"
LINUX_OUT="sfsd_linux_amd64_${VERSION}"

# Build for Windows
echo "Building Windows (amd64)... -> $WIN_OUT"
GOOS=windows GOARCH=amd64 go build -ldflags="-X main.version=$VERSION -s -w" -o "$WIN_OUT" main.go

# Build for Linux
echo "Building Linux (amd64)... -> $LINUX_OUT"
GOOS=linux GOARCH=amd64 go build -ldflags="-X main.version=$VERSION -s -w" -o "$LINUX_OUT" main.go

chmod +x "$LINUX_OUT"

echo "Build Complete: $WIN_OUT, $LINUX_OUT"
