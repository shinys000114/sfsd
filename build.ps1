# Ensure errors stop execution
$ErrorActionPreference = "Stop"

$version = "dev"
$tag = ""
$buildDate = Get-Date -Format "yyMMddHHmmss"

try {
    $isGit = git rev-parse --is-inside-work-tree 2>$null
    if ($isGit -eq "true") {
        try {
            $tag = git describe --tags --abbrev=0 2>$null
        } catch { } # Ignore error if no tags found

        if ([string]::IsNullOrWhiteSpace($tag)) {
            $tag = "dev"
        }
    } else {
        Write-Host "Warning: Not a git repository. Version will be 'dev'."
        $tag = "dev"
    }
} catch {
    Write-Host "Warning: git not found or command failed. Version will be 'dev'."
    $tag = "dev"
}

$version = "$tag+$buildDate"
Write-Host "Building version: $version"

$env:CGO_ENABLED = "0"

Write-Host "Downloading dependencies..."
go mod tidy

$winOut = "sfsd_win_amd64_${version}.exe"
$linuxOut = "sfsd_linux_amd64_${version}"

# Build for Windows
$env:GOOS = "windows"
$env:GOARCH = "amd64"
Write-Host "Building Windows (amd64)... -> $winOut"
go build -ldflags="-X main.version=$version -s -w" -o $winOut main.go

# Build for Linux
$env:GOOS = "linux"
$env:GOARCH = "amd64"
Write-Host "Building Linux (amd64)... -> $linuxOut"
go build -ldflags="-X main.version=$version -s -w" -o $linuxOut main.go

Write-Host "Build Complete: $winOut, $linuxOut"
