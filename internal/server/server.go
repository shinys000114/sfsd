package server

import (
	"fmt"
	"log"
	"net/http"

	"sfsd/internal/config"
	"sfsd/internal/handler"
	"sfsd/internal/middleware"

	"github.com/quic-go/quic-go/http3"
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

		// HTTP/3 (QUIC) support
		if cfg.Server.TLS.HTTP3 {
			originalHandler := handler
			// Add Alt-Svc header to inform clients about HTTP/3 availability
			handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Alt-Svc", fmt.Sprintf(`h3=":%d"; ma=86400`, cfg.Server.Port))
				originalHandler.ServeHTTP(w, r)
			})

			h3Server := &http3.Server{
				Addr:    addr,
				Handler: handler,
			}

			go func() {
				log.Printf("Starting HTTP/3 (QUIC) server on %s", addr)
				if err := h3Server.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile); err != nil {
					log.Printf("HTTP/3 server error: %v", err)
				}
			}()
		}

		return http.ListenAndServeTLS(addr, cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile, handler)
	}

	log.Printf("Starting HTTP server on %s", addr)
	log.Printf("Serving directory: %s", cfg.Directory.Path)
	return http.ListenAndServe(addr, handler)
}
