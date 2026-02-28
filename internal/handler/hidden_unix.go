//go:build !windows
// +build !windows

package handler

import (
	"path/filepath"
	"strings"
)

// isHidden checks if a file is hidden on Unix-like systems (starting with `.`)
func isHidden(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, ".") && base != "." && base != ".."
}
