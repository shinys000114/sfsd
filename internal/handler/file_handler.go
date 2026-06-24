package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"sfsd/internal/config"
)

type FileHandler struct {
	baseDir      string
	cfg          *config.ServerInstance
	excludeRules []excludeRule
}

type servePathContextKey struct{}

// WithServePath returns a request that uses servePath for filesystem lookup.
func WithServePath(r *http.Request, servePath string) *http.Request {
	ctx := context.WithValue(r.Context(), servePathContextKey{}, servePath)
	return r.WithContext(ctx)
}

func servePath(r *http.Request) string {
	if value, ok := r.Context().Value(servePathContextKey{}).(string); ok {
		return value
	}
	return r.URL.Path
}

func NewFileHandler(instanceCfg *config.ServerInstance) *FileHandler {
	absDataDir, err := filepath.Abs(instanceCfg.Directory.Path)
	if err != nil {
		log.Fatalf("Failed to resolve absolute path for serving directory: %v", err)
	}
	baseDir, err := filepath.EvalSymlinks(absDataDir)
	if err != nil {
		log.Fatalf("Failed to resolve serving directory symlinks: %v", err)
	}

	return &FileHandler{
		baseDir:      baseDir,
		cfg:          instanceCfg,
		excludeRules: compileExcludeRules(instanceCfg.Directory.Exclude),
	}
}

func (h *FileHandler) serveCustomError(w http.ResponseWriter, r *http.Request, statusCode int) {
	if h.cfg.Features.Pages != nil {
		if errorPagePath, exists := h.cfg.Features.Pages[statusCode]; exists {
			if _, err := os.Stat(errorPagePath); err == nil {
				w.WriteHeader(statusCode) // explicitly write status code before serving
				http.ServeFile(w, r, errorPagePath)
				return
			}
		}
	}

	var title, msg string
	switch statusCode {
	case http.StatusNotFound:
		title = "404 Not Found"
		msg = "The requested URL was not found on this server."
	case http.StatusForbidden:
		title = "403 Forbidden"
		msg = "You don't have permission to access this resource."
	default:
		title = http.StatusText(statusCode)
		msg = "An error occurred while processing your request."
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>` + title + `</title>
<style>
	body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; text-align: center; padding: 50px; background: #f4f4f4; color: #333; }
	h1 { font-size: 50px; margin-bottom: 10px; }
	p { font-size: 20px; color: #666; }
	hr { border: 0; border-top: 1px solid #ddd; margin: 30px auto; max-width: 400px; }
	.footer { font-size: 14px; color: #999; }
</style>
</head>
<body>
	<h1>` + title + `</h1>
	<p>` + msg + `</p>
	<hr>
	<div class="footer">sfsd - File Server</div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write([]byte(html))
}

func (h *FileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean the requested path to prevent initial path traversal attacks
	cleanPath := filepath.Clean(servePath(r))
	if cleanPath == "/" {
		cleanPath = ""
	}

	// Join with base directory
	fullPath := filepath.Join(h.baseDir, cleanPath)

	// Convert to absolute path to be sure
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		h.serveCustomError(w, r, http.StatusBadRequest)
		return
	}

	// Security: Verify that the resulting absolute path is strictly under the base directory
	if !isPathWithin(h.baseDir, absFullPath) {
		h.serveCustomError(w, r, http.StatusForbidden)
		return
	}

	resolvedPath, fileInfo, statusCode := h.resolvePath(cleanPath)
	if statusCode != 0 {
		h.serveCustomError(w, r, statusCode)
		return
	}

	if isExcludedByRules(h.baseDir, absFullPath, fileInfo.IsDir(), h.excludeRules) ||
		(resolvedPath != absFullPath && isExcludedByRules(h.baseDir, resolvedPath, fileInfo.IsDir(), h.excludeRules)) {
		h.serveCustomError(w, r, http.StatusNotFound)
		return
	}

	// Check if hidden files are allowed
	if h.cfg.Directory.HideHidden {
		if isHiddenPath(h.baseDir, absFullPath) || (resolvedPath != absFullPath && isHiddenPath(h.baseDir, resolvedPath)) {
			// Pretend it doesn't exist
			h.serveCustomError(w, r, http.StatusNotFound)
			return
		}
	}

	if fileInfo.IsDir() {
		// Ensure trailing slash for proper relative link resolution
		if !strings.HasSuffix(r.URL.Path, "/") {
			http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
			return
		}
		serveDirectory(w, r, resolvedPath, r.URL.Path, h.cfg, h.baseDir, h.excludeRules)
		return
	}

	// Set ETag based on file size and modification time for efficient caching
	// Weak ETag is sufficient for file serving and prevents issues with compression
	modTime := fileInfo.ModTime().UnixNano()
	size := fileInfo.Size()
	etag := fmt.Sprintf(`W/"%x-%x"`, size, modTime)
	w.Header().Set("ETag", etag)

	// Serve the file (Range requests and 304 Not Modified are handled automatically by ServeFile)
	http.ServeFile(w, r, resolvedPath)
}

func (h *FileHandler) resolvePath(cleanPath string) (string, os.FileInfo, int) {
	resolvedPath := h.baseDir
	if cleanPath != "" {
		for _, component := range strings.Split(filepath.ToSlash(cleanPath), "/") {
			if component == "" || component == "." {
				continue
			}
			resolvedPath = filepath.Join(resolvedPath, component)
			if !isPathWithin(h.baseDir, resolvedPath) {
				return "", nil, http.StatusForbidden
			}

			fileInfo, err := os.Lstat(resolvedPath)
			if err != nil {
				if os.IsNotExist(err) {
					return "", nil, http.StatusNotFound
				}
				return "", nil, http.StatusInternalServerError
			}

			if fileInfo.Mode()&os.ModeSymlink == 0 {
				continue
			}

			if !h.cfg.Directory.AllowSymlink {
				return "", nil, http.StatusForbidden
			}

			evalPath, err := filepath.EvalSymlinks(resolvedPath)
			if err != nil {
				return "", nil, http.StatusInternalServerError
			}

			absEvalPath, err := filepath.Abs(evalPath)
			if err != nil {
				return "", nil, http.StatusForbidden
			}

			if !h.cfg.Directory.AllowExternalSymlink && !isPathWithin(h.baseDir, absEvalPath) {
				return "", nil, http.StatusForbidden
			}

			resolvedPath = absEvalPath
		}
	}

	fileInfo, err := os.Stat(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, http.StatusNotFound
		}
		return "", nil, http.StatusInternalServerError
	}

	return resolvedPath, fileInfo, 0
}

func isPathWithin(baseDir string, targetPath string) bool {
	rel, err := filepath.Rel(baseDir, targetPath)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel))
}

func isHiddenPath(baseDir string, targetPath string) bool {
	if isPathWithin(baseDir, targetPath) {
		rel, err := filepath.Rel(baseDir, targetPath)
		if err != nil {
			return false
		}
		currentPath := baseDir
		for _, component := range strings.Split(filepath.ToSlash(filepath.Clean(rel)), "/") {
			if component == "." || component == "" {
				continue
			}
			currentPath = filepath.Join(currentPath, component)
			if isHidden(currentPath) || strings.HasPrefix(component, ".") {
				return true
			}
		}
		return false
	}

	return isHidden(targetPath)
}
