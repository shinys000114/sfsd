# Ensure errors stop execution
$ErrorActionPreference = "Stop"

$version = "dev"
$tag = ""
$commitHash = ""

try {
    $isGit = git rev-parse --is-inside-work-tree 2>$null
    if ($isGit -eq "true") {
        $commitHash = git rev-parse --short HEAD 2>$null

        try {
            $tag = git describe --tags --abbrev=0 2>$null
        } catch { } # Ignore error if no tags found

        if ([string]::IsNullOrWhiteSpace($commitHash)) {
            $version = "dev"
        } elseif ([string]::IsNullOrWhiteSpace($tag)) {
            $version = "dev-$commitHash"
            $tag = "dev"
        } else {
            $version = "$tag-$commitHash"
        }
    } else {
        Write-Host "Warning: Not a git repository. Version will be 'dev'."
        $tag = "dev"
        $commitHash = "nohash"
    }
} catch {
    Write-Host "Warning: git not found or command failed. Version will be 'dev'."
    $tag = "dev"
    $commitHash = "nohash"
}

Write-Host "Building version: $version"

$env:CGO_ENABLED = "0"

$winOut = "sfsd_win_amd64_${tag}_${commitHash}.exe"
$linuxOut = "sfsd_linux_amd64_${tag}_${commitHash}"

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