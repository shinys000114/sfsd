#!/bin/bash

VERSION="dev"
TAG=""
COMMIT_HASH=""

if ! git rev-parse --is-inside-work-tree > /dev/null 2>&1; then
    echo "Warning: Not a git repository. Version will be 'dev'."
    TAG="dev"
    COMMIT_HASH="nohash"
else
    COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null)
    TAG=$(git describe --tags --abbrev=0 2>/dev/null)

    if [ -z "$COMMIT_HASH" ]; then
        VERSION="dev"
        TAG="dev"
        COMMIT_HASH="nohash"
    elif [ -z "$TAG" ]; then
        VERSION="dev-$COMMIT_HASH"
        TAG="dev"
    else
        VERSION="$TAG-$COMMIT_HASH"
    fi
fi

echo "Building version: $VERSION"

export CGO_ENABLED=0

WIN_OUT="sfsd_win_amd64_${TAG}_${COMMIT_HASH}.exe"
LINUX_OUT="sfsd_linux_amd64_${TAG}_${COMMIT_HASH}"

# Build for Windows
echo "Building Windows (amd64)... -> $WIN_OUT"
GOOS=windows GOARCH=amd64 go build -ldflags="-X main.version=$VERSION -s -w" -o "$WIN_OUT" main.go

# Build for Linux
echo "Building Linux (amd64)... -> $LINUX_OUT"
GOOS=linux GOARCH=amd64 go build -ldflags="-X main.version=$VERSION -s -w" -o "$LINUX_OUT" main.go

chmod +x "$LINUX_OUT"

echo "Build Complete: $WIN_OUT, $LINUX_OUT"