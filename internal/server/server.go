package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/quic-go/quic-go/http3"
	"sfsd/internal/config"
	"sfsd/internal/handler"
	"sfsd/internal/middleware"
)

// Start initializes and starts a single server instance
func Start(name string, cfg *config.ServerInstance) error {
	fileHandler := handler.NewFileHandler(cfg)

	mux := http.NewServeMux()
	// All requests go to the file handler
	mux.Handle("/", fileHandler)

	// Build middleware chain (from inside out)
	var h http.Handler = mux

	// 1. Core security & features
	h = middleware.DownloadCounter(h)
	h = middleware.Cache(cfg.Features.CacheRules, h)
	h = middleware.Compress(cfg.Features.Compression, h)
	h = middleware.CORS(cfg.Features.CORSEnabled, h)

	// 2. Auth (if enabled, protects everything)
	h = middleware.BasicAuth(cfg.Features.Auth, h)

	// 3. Outermost Logging
	h = middleware.Logger(cfg, h)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	serveTLS := cfg.Server.TLS.Enabled

	if serveTLS {
		// HTTP/3 (QUIC) support
		if cfg.Server.TLS.HTTP3 {
			originalHandler := h
			// Add Alt-Svc header to inform clients about HTTP/3 availability
			h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Alt-Svc", fmt.Sprintf(`h3=":%d"; ma=86400`, cfg.Server.Port))
				originalHandler.ServeHTTP(w, r)
			})

			h3Server := &http3.Server{
				Addr:    addr,
				Handler: h,
			}

			go func() {
				log.Printf("[%s] Starting HTTP/3 (QUIC) server on %s", name, addr)
				if err := h3Server.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile); err != nil {
					log.Printf("[%s] HTTP/3 server error: %v", name, err)
				}
			}()
		}

		log.Printf("[%s] Starting HTTPS server on %s", name, addr)
		return http.ListenAndServeTLS(addr, cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile, h)
	}

	log.Printf("[%s] Starting HTTP server on %s", name, addr)
	return http.ListenAndServe(addr, h)
}
