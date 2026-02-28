#!/bin/bash

if ! git rev-parse --is-inside-work-tree > /dev/null 2>&1; then
    echo "Warning: Not a git repository. Version will be 'dev'."
    VERSION="dev"
else
    COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null)
    TAG=$(git describe --tags --abbrev=0 2>/dev/null)
    
    if [ -z "$COMMIT_HASH" ]; then
        VERSION="dev"
    elif [ -z "$TAG" ]; then
        VERSION="dev-$COMMIT_HASH"
    else
        VERSION="$TAG-$COMMIT_HASH"
    fi
fi

echo "Building version: $VERSION"

export CGO_ENABLED=0

# Build for Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-X main.version=$VERSION -s -w" -o sfsd.exe main.go

# Build for Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-X main.version=$VERSION -s -w" -o sfsd main.go

echo "Build Complete: sfsd.exe, sfsd"
