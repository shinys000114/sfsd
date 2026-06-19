package handler

import (
	"path"
	"path/filepath"
	"regexp"
	"sfsd/internal/pattern"
	"strings"
)

type excludeRule struct {
	negated  bool
	dirOnly  bool
	anchored bool
	hasSlash bool
	regex    *regexp.Regexp
}

func compileExcludeRules(patterns []string) []excludeRule {
	rules := make([]excludeRule, 0, len(patterns))

	for _, raw := range patterns {
		excludePattern := strings.TrimSpace(raw)
		if excludePattern == "" || strings.HasPrefix(excludePattern, "#") {
			continue
		}

		negated := strings.HasPrefix(excludePattern, "!")
		if negated {
			excludePattern = strings.TrimSpace(strings.TrimPrefix(excludePattern, "!"))
		}
		if excludePattern == "" {
			continue
		}

		dirOnly := strings.HasSuffix(excludePattern, "/")
		excludePattern = filepath.ToSlash(strings.TrimSuffix(excludePattern, "/"))
		anchored := strings.HasPrefix(excludePattern, "/")
		excludePattern = strings.TrimPrefix(excludePattern, "/")
		excludePattern = strings.TrimPrefix(excludePattern, "./")
		excludePattern = path.Clean(excludePattern)
		if excludePattern == "." {
			continue
		}

		regex, err := regexp.Compile("^" + pattern.GlobToRegexp(excludePattern) + "$")
		if err != nil {
			continue
		}

		rules = append(rules, excludeRule{
			negated:  negated,
			dirOnly:  dirOnly,
			anchored: anchored,
			hasSlash: strings.Contains(excludePattern, "/"),
			regex:    regex,
		})
	}

	return rules
}

func isExcludedByRules(baseDir string, fullPath string, isDir bool, rules []excludeRule) bool {
	if len(rules) == 0 {
		return false
	}

	rel, err := filepath.Rel(baseDir, fullPath)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
		return false
	}

	excluded := false
	for _, rule := range rules {
		if rule.matches(rel, isDir) {
			excluded = !rule.negated
		}
	}
	return excluded
}

func (r excludeRule) matches(rel string, isDir bool) bool {
	if r.anchored || r.hasSlash {
		if r.dirOnly {
			return r.matchesDirectory(rel)
		}
		return r.regex.MatchString(rel)
	}

	parts := strings.Split(rel, "/")
	for i, part := range parts {
		if !r.regex.MatchString(part) {
			continue
		}
		if !r.dirOnly || isDir || i < len(parts)-1 {
			return true
		}
	}
	return false
}

func (r excludeRule) matchesDirectory(rel string) bool {
	current := rel
	for {
		if r.regex.MatchString(current) {
			return true
		}
		idx := strings.LastIndex(current, "/")
		if idx < 0 {
			return false
		}
		current = current[:idx]
	}
}
