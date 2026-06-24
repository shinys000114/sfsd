package server

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/quic-go/quic-go/http3"
	"sfsd/internal/config"
	"sfsd/internal/handler"
	"sfsd/internal/middleware"
)

type vhostRoute struct {
	pathPrefix string
	handler    http.Handler
}

// VHostHandler routes requests based on the Host header and optional path prefix.
type VHostHandler struct {
	hosts          map[string][]vhostRoute
	defaultHandler http.Handler
}

func (vh *VHostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := normalizeHost(r.Host)
	routes, hostExists := vh.hosts[host]
	requestPath := normalizeRoutePath(r.URL.Path)

	var matched *vhostRoute
	for i := range routes {
		route := &routes[i]
		if routeMatchesPath(route.pathPrefix, requestPath) &&
			(matched == nil || len(route.pathPrefix) > len(matched.pathPrefix)) {
			matched = route
		}
	}

	if matched != nil {
		matched.handler.ServeHTTP(w, r)
		return
	}
	if hostExists {
		http.Error(w, "Path not found", http.StatusNotFound)
		return
	}
	if vh.defaultHandler != nil {
		vh.defaultHandler.ServeHTTP(w, r)
		return
	}

	http.Error(w, "Host not found", http.StatusNotFound)
}

// StartGroup initializes and starts multiple server instances on a single address
func StartGroup(addr string, instances map[string]*config.ServerInstance) error {
	vhost := &VHostHandler{
		hosts: make(map[string][]vhostRoute),
	}

	certs := make(map[string]tls.Certificate)
	certSources := make(map[string]string)
	var firstCert *tls.Certificate
	var http3Enabled bool
	var tlsEnabled bool
	var tlsModeSet bool

	for name, cfg := range instances {
		if !tlsModeSet {
			tlsEnabled = cfg.Server.TLS.Enabled
			tlsModeSet = true
		} else if tlsEnabled != cfg.Server.TLS.Enabled {
			return fmt.Errorf("[%s] cannot mix TLS and non-TLS instances on %s", name, addr)
		}

		fileHandler := handler.NewFileHandler(cfg)
		mux := http.NewServeMux()
		mux.Handle("/", fileHandler)

		var h http.Handler = mux
		if counter := middleware.NewCounter(cfg.Features.StatsFile); counter != nil {
			counter.StartAutoSave()
			h = counter.DownloadCounter(h)
		}
		h = middleware.Cache(cfg.Features.CacheRules, h)
		h = middleware.Compress(cfg.Features.Compression, h)
		h = middleware.CORS(cfg.Features.CORSEnabled, h)
		h = middleware.BasicAuth(cfg.Features.Auth, h)
		h = middleware.Logger(cfg, h)

		// Register all domains for this instance
		if len(cfg.Server.Domains) > 0 {
			for _, domain := range cfg.Server.Domains {
				host, pathPrefix, err := parseDomainRoute(domain)
				if err != nil {
					return fmt.Errorf("[%s] invalid vhost domain %q: %w", name, domain, err)
				}
				for _, route := range vhost.hosts[host] {
					if route.pathPrefix == pathPrefix {
						return fmt.Errorf("[%s] duplicate vhost route: %s%s", name, host, displayRoutePath(pathPrefix))
					}
				}
				vhost.hosts[host] = append(vhost.hosts[host], vhostRoute{
					pathPrefix: pathPrefix,
					handler:    h,
				})
			}
		} else {
			// If no domains, we can't reliably route multiple instances on same port
			// but we'll use it as a "default" for this address if it's the only one.
			if vhost.defaultHandler != nil {
				return fmt.Errorf("[%s] multiple default instances configured for %s", name, addr)
			}
			vhost.defaultHandler = h
		}

		// Prepare TLS if enabled
		if cfg.Server.TLS.Enabled {
			cert, err := tls.LoadX509KeyPair(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
			if err != nil {
				return fmt.Errorf("[%s] Failed to load TLS cert: %v", name, err)
			}

			if firstCert == nil {
				firstCert = &cert
			}

			certSource := cfg.Server.TLS.CertFile + "\x00" + cfg.Server.TLS.KeyFile
			for _, domain := range cfg.Server.Domains {
				host, _, err := parseDomainRoute(domain)
				if err != nil {
					return fmt.Errorf("[%s] invalid vhost domain %q: %w", name, domain, err)
				}
				if source, exists := certSources[host]; exists && source != certSource {
					return fmt.Errorf("[%s] multiple TLS certificates configured for vhost domain: %s", name, host)
				}
				certSources[host] = certSource
				certs[host] = cert
			}

			if cfg.Server.TLS.HTTP3 {
				http3Enabled = true
			}
		}
	}

	// Keep the direct handler optimization for a single host-wide or default instance.
	var finalHandler http.Handler = vhost
	if len(vhost.hosts) == 0 && vhost.defaultHandler != nil {
		finalHandler = vhost.defaultHandler
	} else if len(vhost.hosts) == 1 && vhost.defaultHandler == nil {
		for _, routes := range vhost.hosts {
			if len(routes) == 1 && routes[0].pathPrefix == "/" {
				finalHandler = routes[0].handler
			}
		}
	}

	if tlsEnabled {
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{*firstCert},
			GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
				if cert, ok := certs[normalizeHost(info.ServerName)]; ok {
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
			altSvcPort, err := portFromAddr(addr)
			if err != nil {
				return err
			}

			h3Server := &http3.Server{
				Addr:      addr,
				Handler:   finalHandler,
				TLSConfig: tlsConfig,
			}

			// Add Alt-Svc header
			h := finalHandler
			finalHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Alt-Svc", fmt.Sprintf(`h3=":%d"; ma=86400`, altSvcPort))
				h.ServeHTTP(w, r)
			})
			h3Server.Handler = finalHandler
			httpServer.Handler = finalHandler

			go func() {
				log.Printf("Starting HTTP/3 (QUIC) server on %s", addr)
				if err := h3Server.ListenAndServe(); err != nil {
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

func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}

	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	} else if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	}

	host = strings.TrimSuffix(host, ".")
	return strings.ToLower(host)
}

func parseDomainRoute(domain string) (string, string, error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return "", "", fmt.Errorf("empty domain")
	}

	hostPart := domain
	pathPrefix := "/"
	if slash := strings.IndexByte(domain, '/'); slash >= 0 {
		hostPart = domain[:slash]
		pathPrefix = normalizeRoutePath(domain[slash:])
	}

	host := normalizeHost(hostPart)
	if host == "" {
		return "", "", fmt.Errorf("empty host")
	}
	if strings.ContainsAny(pathPrefix, "?#") {
		return "", "", fmt.Errorf("path must not contain a query or fragment")
	}

	return host, pathPrefix, nil
}

func normalizeRoutePath(routePath string) string {
	return path.Clean("/" + strings.TrimPrefix(routePath, "/"))
}

func routeMatchesPath(pathPrefix, requestPath string) bool {
	if pathPrefix == "/" {
		return true
	}
	return requestPath == pathPrefix || strings.HasPrefix(requestPath, pathPrefix+"/")
}

func displayRoutePath(pathPrefix string) string {
	if pathPrefix == "/" {
		return ""
	}
	return pathPrefix
}

func portFromAddr(addr string) (int, error) {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return 0, fmt.Errorf("invalid listen address %q: %w", addr, err)
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil {
		return 0, fmt.Errorf("invalid listen port %q: %w", port, err)
	}
	if portNumber <= 0 || portNumber > 65535 {
		return 0, fmt.Errorf("listen port out of range: %d", portNumber)
	}

	return portNumber, nil
}
