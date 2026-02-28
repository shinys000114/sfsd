package middleware

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sfsd/internal/config"
)

func Cache(rules []config.CacheRule, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		maxAge := -1

		// Evaluate rules in order
		for _, rule := range rules {
			matched, err := filepath.Match(rule.Pattern, filepath.Base(r.URL.Path))
			if err == nil && matched {
				maxAge = rule.MaxAge
				break
			}
		}

		if maxAge >= 0 {
			w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
		} else {
			// fallback no-cache if no rules map
			w.Header().Set("Cache-Control", "no-cache")
		}

		next.ServeHTTP(w, r)
	})
}
