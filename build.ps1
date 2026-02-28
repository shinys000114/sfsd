# Ensure errors stop execution
$ErrorActionPreference = "Stop"

$version = "dev"
try {
    $version = git describe --tags --always --dirty 2>$null
    if ($LASTEXITCODE -ne 0) { $version = "dev" }
} catch {
    Write-Host "Warning: git not found or not a git repository. Version will be 'dev'."
}

Write-Host "Building version: $version"

$env:CGO_ENABLED = "0"

# Build for Windows
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -ldflags="-X main.version=$version -s -w" -o sfsd.exe main.go

# Build for Linux
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -ldflags="-X main.version=$version -s -w" -o sfsd main.go

Write-Host "Build Complete: sfsd.exe, sfsd"
