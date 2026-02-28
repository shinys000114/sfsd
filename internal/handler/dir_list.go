package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"

	"sfsd/internal/config"
)

type FileInfo struct {
	Name    string
	URL     string
	IsDir   bool
	Size    string
	ModTime string
}

type DirData struct {
	Path  string
	Files []FileInfo
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// serveDirectory reads custom directory listing and writes HTML
func serveDirectory(w http.ResponseWriter, r *http.Request, fullPath string, reqPath string, cfg *config.Config) {
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var files []FileInfo
	for _, entry := range entries {
		// Skip hidden files if configured
		if cfg.Directory.HideHidden && isHidden(path.Join(fullPath, entry.Name())) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		sizeStr := "-"
		if !entry.IsDir() {
			sizeStr = formatSize(info.Size())
		}

		// URL needs to be properly path joined and escaped but simple path.Join works since it's standard HTTP
		urlName := entry.Name()
		if entry.IsDir() {
			urlName += "/"
		}

		files = append(files, FileInfo{
			Name:    entry.Name(),
			URL:     urlName,
			IsDir:   entry.IsDir(),
			Size:    sizeStr,
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	// Sort: Directories first, then alphabetically
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir && !files[j].IsDir {
			return true
		}
		if !files[i].IsDir && files[j].IsDir {
			return false
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	data := DirData{
		Path:  reqPath,
		Files: files,
	}

	tmpl, err := template.New("dir").Parse(defaultDirTemplate)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Always disable cache for directory index pages to show real-time changes
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(http.StatusOK)
	if err := tmpl.Execute(w, data); err != nil {
		// Log error but body is already started
	}
}
