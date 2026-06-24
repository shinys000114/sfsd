package handler

import (
	"mime"
	"path/filepath"
	"sort"
	"strings"

	"sfsd/internal/config"
)

const (
	defaultDirectoryIcon = "📁"
	defaultFileIcon      = "📄"
)

type extensionIcon struct {
	suffix string
	icon   string
}

type iconResolver struct {
	directoryIcon string
	defaultIcon   string
	extensions    []extensionIcon
	mimeTypes     map[string]string
}

func newIconResolver(cfg config.IconConfig) iconResolver {
	resolver := iconResolver{
		directoryIcon: cfg.Directory,
		defaultIcon:   cfg.Default,
		mimeTypes:     make(map[string]string, len(cfg.MIMETypes)),
	}
	if resolver.directoryIcon == "" {
		resolver.directoryIcon = defaultDirectoryIcon
	}
	if resolver.defaultIcon == "" {
		resolver.defaultIcon = defaultFileIcon
	}

	for extension, icon := range cfg.Extensions {
		extension = strings.ToLower(strings.TrimSpace(extension))
		if extension == "" || icon == "" {
			continue
		}
		if !strings.HasPrefix(extension, ".") {
			extension = "." + extension
		}
		resolver.extensions = append(resolver.extensions, extensionIcon{
			suffix: extension,
			icon:   icon,
		})
	}
	sort.Slice(resolver.extensions, func(i, j int) bool {
		return len(resolver.extensions[i].suffix) > len(resolver.extensions[j].suffix)
	})

	for mimeType, icon := range cfg.MIMETypes {
		mimeType = strings.ToLower(strings.TrimSpace(mimeType))
		if mimeType != "" && icon != "" {
			resolver.mimeTypes[mimeType] = icon
		}
	}

	return resolver
}

func (r iconResolver) iconFor(name string, isDir bool) string {
	if isDir {
		return r.directoryIcon
	}

	lowerName := strings.ToLower(name)
	for _, extension := range r.extensions {
		if strings.HasSuffix(lowerName, extension.suffix) {
			return extension.icon
		}
	}

	extension := strings.ToLower(filepath.Ext(name))
	mimeType := strings.ToLower(mime.TypeByExtension(extension))
	if separator := strings.IndexByte(mimeType, ';'); separator >= 0 {
		mimeType = mimeType[:separator]
	}
	if icon, ok := r.mimeTypes[mimeType]; ok {
		return icon
	}
	if separator := strings.IndexByte(mimeType, '/'); separator >= 0 {
		if icon, ok := r.mimeTypes[mimeType[:separator]+"/*"]; ok {
			return icon
		}
	}

	return r.defaultIcon
}
