package handler

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"sfsd/internal/config"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var mdParser = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		highlighting.NewHighlighting(
			highlighting.WithStyle("github"),
		),
		extension.Footnote,
		extension.Typographer,
		extension.DefinitionList,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		html.WithXHTML(),
	),
)

type FileInfo struct {
	Name    string
	URL     string
	Icon    string
	IsDir   bool
	Size    string
	ModTime string
}

type DirData struct {
	Path          string
	DirectoryIcon string
	Files         []FileInfo
	MdHtml        template.HTML
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
func serveDirectory(w http.ResponseWriter, r *http.Request, fullPath string, reqPath string, cfg *config.ServerInstance, baseDir string, excludeRules []excludeRule) {
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var files []FileInfo
	var readmeContent []byte
	icons := newIconResolver(cfg.Directory.Icons)

	for _, entry := range entries {
		name := entry.Name()
		entryPath := filepath.Join(fullPath, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}
		isDir := info.IsDir()

		// Skip hidden files
		if cfg.Directory.HideHidden && isHiddenPath(baseDir, entryPath) {
			continue
		}

		if entry.Type()&os.ModeSymlink != 0 {
			if !cfg.Directory.AllowSymlink {
				continue
			}
			resolvedPath, err := filepath.EvalSymlinks(entryPath)
			if err != nil {
				continue
			}
			if !cfg.Directory.AllowExternalSymlink && !isPathWithin(baseDir, resolvedPath) {
				continue
			}
			resolvedInfo, err := os.Stat(resolvedPath)
			if err != nil {
				continue
			}
			info = resolvedInfo
			isDir = resolvedInfo.IsDir()
			if cfg.Directory.HideHidden && isHiddenPath(baseDir, resolvedPath) {
				continue
			}
			if isExcludedByRules(baseDir, resolvedPath, resolvedInfo.IsDir(), excludeRules) {
				continue
			}
		}
		if isExcludedByRules(baseDir, entryPath, isDir, excludeRules) {
			continue
		}

		if cfg.Directory.RenderReadmeMd && !isDir && readmeContent == nil {
			if strings.EqualFold(name, "readme.md") {
				readmeContent, _ = os.ReadFile(entryPath)
			}
		}

		sizeStr := "-"
		if !isDir {
			sizeStr = formatSize(info.Size())
		}

		urlName := name
		if isDir {
			urlName += "/"
		}

		files = append(files, FileInfo{
			Name:    name,
			URL:     urlName,
			Icon:    icons.iconFor(name, isDir),
			IsDir:   isDir,
			Size:    sizeStr,
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	// Sort: Directories first, then alphabetically
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	var mdHtml template.HTML
	if len(readmeContent) > 0 {
		var buf bytes.Buffer
		if err := mdParser.Convert(readmeContent, &buf); err == nil {
			mdHtml = template.HTML(buf.String())
		}
	}

	data := DirData{
		Path:          reqPath,
		DirectoryIcon: icons.directoryIcon,
		Files:         files,
		MdHtml:        mdHtml,
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
