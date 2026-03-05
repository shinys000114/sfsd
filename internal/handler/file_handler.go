package handler

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"sfsd/internal/config"
)

type FileHandler struct {
	baseDir string
	cfg     *config.ServerInstance
}

func NewFileHandler(instanceCfg *config.ServerInstance) *FileHandler {
	absDataDir, err := filepath.Abs(instanceCfg.Directory.Path)
	if err != nil {
		log.Fatalf("Failed to resolve absolute path for serving directory: %v", err)
	}

	return &FileHandler{
		baseDir: absDataDir,
		cfg:     instanceCfg,
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
	cleanPath := filepath.Clean(r.URL.Path)
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
	if !strings.HasPrefix(absFullPath, h.baseDir) {
		h.serveCustomError(w, r, http.StatusForbidden)
		return
	}

	// Resolve Symlink if needed
	resolvedPath := absFullPath
	fileInfo, err := os.Lstat(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			h.serveCustomError(w, r, http.StatusNotFound)
			return
		}
		h.serveCustomError(w, r, http.StatusInternalServerError)
		return
	}

	if fileInfo.Mode()&os.ModeSymlink != 0 {
		if !h.cfg.Directory.AllowSymlink {
			h.serveCustomError(w, r, http.StatusForbidden)
			return
		}

		evalPath, err := filepath.EvalSymlinks(resolvedPath)
		if err != nil {
			h.serveCustomError(w, r, http.StatusInternalServerError)
			return
		}

		absEvalPath, err := filepath.Abs(evalPath)
		if err != nil {
			h.serveCustomError(w, r, http.StatusForbidden)
			return
		}

		if !h.cfg.Directory.AllowExternalSymlink && !strings.HasPrefix(absEvalPath, h.baseDir) {
			h.serveCustomError(w, r, http.StatusForbidden)
			return
		}

		// Update resolved path for serving
		resolvedPath = absEvalPath
		fileInfo, err = os.Stat(resolvedPath) // get actual file info
		if err != nil {
			h.serveCustomError(w, r, http.StatusInternalServerError)
			return
		}
	}

	// Check if hidden files are allowed
	if h.cfg.Directory.HideHidden {
		if isHidden(resolvedPath) {
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
		serveDirectory(w, r, resolvedPath, r.URL.Path, h.cfg)
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
