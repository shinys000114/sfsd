package middleware

import (
	"fmt"
	"log"
	"net/http"
	"sfsd/internal/config"
	"sfsd/internal/pattern"
	"strings"
)

type compiledCacheRule struct {
	Glob   *pattern.Glob
	MaxAge int
}

func Cache(rules []config.CacheRule, next http.Handler) http.Handler {
	var compiledRules []compiledCacheRule

	for _, rule := range rules {
		compiled, err := pattern.CompileGlob(rule.Pattern)
		if err != nil {
			log.Printf("Warning: Invalid cache rule glob pattern '%s': %v\n", rule.Pattern, err)
			continue
		}

		compiledRules = append(compiledRules, compiledCacheRule{
			Glob:   compiled,
			MaxAge: rule.MaxAge,
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		maxAge := -1
		reqPath := strings.TrimPrefix(r.URL.Path, "/")
		if reqPath == "" {
			reqPath = "/"
		}

		for _, crule := range compiledRules {
			if crule.matches(reqPath) {
				maxAge = crule.MaxAge
				break
			}
		}

		if maxAge >= 0 {
			w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}

		next.ServeHTTP(w, r)
	})
}

func (r compiledCacheRule) matches(reqPath string) bool {
	return r.Glob.MatchPath(reqPath)
}
