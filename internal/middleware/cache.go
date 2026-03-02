package middleware

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"sfsd/internal/config"
)

type compiledCacheRule struct {
	Regex  *regexp.Regexp
	MaxAge int
}

func Cache(rules []config.CacheRule, next http.Handler) http.Handler {
	var compiledRules []compiledCacheRule

	for _, rule := range rules {
		compiled, err := regexp.Compile(rule.Pattern)
		if err != nil {
			log.Printf("Warning: Invalid cache rule regex pattern '%s': %v\n", rule.Pattern, err)
			continue
		}

		compiledRules = append(compiledRules, compiledCacheRule{
			Regex:  compiled,
			MaxAge: rule.MaxAge,
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		maxAge := -1
		baseName := filepath.Base(r.URL.Path)

		for _, crule := range compiledRules {
			if crule.Regex.MatchString(baseName) {
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
