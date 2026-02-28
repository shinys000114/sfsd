# Ensure errors stop execution
$ErrorActionPreference = "Stop"

$version = "dev"
try {
    $isGit = git rev-parse --is-inside-work-tree 2>$null
    if ($isGit -eq "true") {
        $commitHash = git rev-parse --short HEAD 2>$null
        
        $tag = ""
        try {
            $tag = git describe --tags --abbrev=0 2>$null
        } catch { } # Ignore error if no tags found

        if ([string]::IsNullOrWhiteSpace($commitHash)) {
            $version = "dev"
        } elseif ([string]::IsNullOrWhiteSpace($tag)) {
            $version = "dev-$commitHash"
        } else {
            $version = "$tag-$commitHash"
        }
    } else {
        Write-Host "Warning: Not a git repository. Version will be 'dev'."
    }
} catch {
    Write-Host "Warning: git not found or command failed. Version will be 'dev'."
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
