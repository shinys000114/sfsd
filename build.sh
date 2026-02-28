#!/bin/bash

# Ensure git is initialized and tag exists
if ! git rev-parse --is-inside-work-tree > /dev/null 2>&1; then
    echo "Warning: Not a git repository. Version will be 'dev'."
    VERSION="dev"
else
    VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
fi

echo "Building version: $VERSION"

export CGO_ENABLED=0

# Build for Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-X main.version=$VERSION -s -w" -o sfsd.exe main.go

# Build for Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-X main.version=$VERSION -s -w" -o sfsd main.go

echo "Build Complete: sfsd.exe, sfsd"
