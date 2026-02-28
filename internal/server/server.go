package server

import (
	"fmt"
	"log"
	"net/http"

	"sfsd/internal/config"
	"sfsd/internal/handler"
	"sfsd/internal/middleware"
)

// Start initializes and starts the HTTP(S) server
func Start(cfg *config.Config) error {
	fileHandler := handler.NewFileHandler(cfg)

	mux := http.NewServeMux()
	// All requests go to the file handler
	mux.Handle("/", fileHandler)

	// Build middleware chain (from inside out)
	var handler http.Handler = mux

	// 1. Core security & features
	handler = middleware.DownloadCounter(handler)
	handler = middleware.Cache(cfg.Features.CacheRules, handler)
	handler = middleware.Compress(cfg.Features.Compression, handler)
	handler = middleware.CORS(cfg.Features.CORSEnabled, handler)

	// 2. Auth (if enabled, protects everything)
	handler = middleware.BasicAuth(cfg.Features.Auth, handler)

	// 3. Outermost Logging
	handler = middleware.Logger(cfg, handler)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	serveTLS := cfg.Server.TLS.Enabled

	if serveTLS {
		log.Printf("Starting HTTPS server on %s", addr)
		log.Printf("Serving directory: %s", cfg.Directory.Path)
		return http.ListenAndServeTLS(addr, cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile, handler)
	}

	log.Printf("Starting HTTP server on %s", addr)
	log.Printf("Serving directory: %s", cfg.Directory.Path)
	return http.ListenAndServe(addr, handler)
}
