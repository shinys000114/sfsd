package server

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/quic-go/quic-go/http3"
	"sfsd/internal/config"
	"sfsd/internal/handler"
	"sfsd/internal/middleware"
)

// VHostHandler routes requests based on the Host header
type VHostHandler struct {
	hosts map[string]http.Handler
}

func (vh *VHostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	// Remove port from host if present
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	if h, ok := vh.hosts[host]; ok {
		h.ServeHTTP(w, r)
		return
	}

	// Default to the first configured host if no match (optional)
	// Or serve 404
	http.Error(w, "Host not found", http.StatusNotFound)
}

// StartGroup initializes and starts multiple server instances on a single address
func StartGroup(addr string, instances map[string]*config.ServerInstance) error {
	vhost := &VHostHandler{
		hosts: make(map[string]http.Handler),
	}

	certs := make(map[string]tls.Certificate)
	var firstCert *tls.Certificate
	var http3Enabled bool
	var tlsEnabled bool

	for name, cfg := range instances {
		fileHandler := handler.NewFileHandler(cfg)
		mux := http.NewServeMux()
		mux.Handle("/", fileHandler)

		var h http.Handler = mux
		h = middleware.DownloadCounter(h)
		h = middleware.Cache(cfg.Features.CacheRules, h)
		h = middleware.Compress(cfg.Features.Compression, h)
		h = middleware.CORS(cfg.Features.CORSEnabled, h)
		h = middleware.BasicAuth(cfg.Features.Auth, h)
		h = middleware.Logger(cfg, h)

		// Register all domains for this instance
		if len(cfg.Server.Domains) > 0 {
			for _, domain := range cfg.Server.Domains {
				vhost.hosts[domain] = h
			}
		} else {
			// If no domains, we can't reliably route multiple instances on same port
			// but we'll use it as a "default" for this address if it's the only one.
			vhost.hosts["_default_"] = h
		}

		// Prepare TLS if enabled
		if cfg.Server.TLS.Enabled {
			tlsEnabled = true
			cert, err := tls.LoadX509KeyPair(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
			if err != nil {
				return fmt.Errorf("[%s] Failed to load TLS cert: %v", name, err)
			}

			if firstCert == nil {
				firstCert = &cert
			}

			for _, domain := range cfg.Server.Domains {
				certs[domain] = cert
			}

			if cfg.Server.TLS.HTTP3 {
				http3Enabled = true
			}
		}
	}

	// Final handler: if only one instance with no specific domains, use it directly
	var finalHandler http.Handler = vhost
	if len(vhost.hosts) == 1 {
		for _, h := range vhost.hosts {
			finalHandler = h
		}
	} else if _, ok := vhost.hosts["_default_"]; ok && len(vhost.hosts) > 1 {
		// If we have mixed named and unnamed hosts, we might need a better fallback
		defaultHandler := vhost.hosts["_default_"]
		originalVHost := vhost.ServeHTTP
		finalHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := r.Host
			if h, _, err := net.SplitHostPort(host); err == nil {
				host = h
			}
			if _, ok := vhost.hosts[host]; ok {
				originalVHost(w, r)
			} else {
				defaultHandler.ServeHTTP(w, r)
			}
		})
	}

	if tlsEnabled {
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{*firstCert},
			GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
				if cert, ok := certs[info.ServerName]; ok {
					return &cert, nil
				}
				return firstCert, nil
			},
			NextProtos: []string{"h2", "http/1.1"},
		}

		httpServer := &http.Server{
			Addr:      addr,
			Handler:   finalHandler,
			TLSConfig: tlsConfig,
		}

		if http3Enabled {
			h3Server := &http3.Server{
				Addr:    addr,
				Handler: finalHandler,
			}

			// Add Alt-Svc header
			h := finalHandler
			finalHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Alt-Svc", fmt.Sprintf(`h3=":%s"; ma=86400`, strings.Split(addr, ":")[1]))
				h.ServeHTTP(w, r)
			})
			h3Server.Handler = finalHandler
			httpServer.Handler = finalHandler

			go func() {
				log.Printf("Starting HTTP/3 (QUIC) server on %s", addr)
				if err := h3Server.ListenAndServeTLS(instances[getFirstKey(instances)].Server.TLS.CertFile, instances[getFirstKey(instances)].Server.TLS.KeyFile); err != nil {
					log.Printf("HTTP/3 server error: %v", err)
				}
			}()
		}

		log.Printf("Starting HTTPS server on %s (Shared port for %d instances)", addr, len(instances))
		return httpServer.ListenAndServeTLS("", "") // certs are in tlsConfig
	}

	log.Printf("Starting HTTP server on %s (Shared port for %d instances)", addr, len(instances))
	return http.ListenAndServe(addr, finalHandler)
}

func getFirstKey(m map[string]*config.ServerInstance) string {
	for k := range m {
		return k
	}
	return ""
}
