//go:build windows
// +build windows

package handler

import (
	"path/filepath"
	"strings"
	"syscall"
)

// isHidden checks if a file is hidden on Windows (both starting with `.` and having the Hidden attribute)
func isHidden(path string) bool {
	// 1. Check if name starts with "." (Unix-like hidden file convention)
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".") && base != "." && base != ".." {
		return true
	}

	// 2. Check Windows file attributes
	pointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false // On error, allow it (or block it? default allow and let other checks fail)
	}
	attributes, err := syscall.GetFileAttributes(pointer)
	if err != nil {
		return false
	}
	return attributes&syscall.FILE_ATTRIBUTE_HIDDEN != 0
}
