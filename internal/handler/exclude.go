package handler

import (
	"path"
	"path/filepath"
	"regexp"
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
		pattern := strings.TrimSpace(raw)
		if pattern == "" || strings.HasPrefix(pattern, "#") {
			continue
		}

		negated := strings.HasPrefix(pattern, "!")
		if negated {
			pattern = strings.TrimSpace(strings.TrimPrefix(pattern, "!"))
		}
		if pattern == "" {
			continue
		}

		dirOnly := strings.HasSuffix(pattern, "/")
		pattern = filepath.ToSlash(strings.TrimSuffix(pattern, "/"))
		anchored := strings.HasPrefix(pattern, "/")
		pattern = strings.TrimPrefix(pattern, "/")
		pattern = strings.TrimPrefix(pattern, "./")
		pattern = path.Clean(pattern)
		if pattern == "." {
			continue
		}

		regex, err := regexp.Compile("^" + globToRegexp(pattern) + "$")
		if err != nil {
			continue
		}

		rules = append(rules, excludeRule{
			negated:  negated,
			dirOnly:  dirOnly,
			anchored: anchored,
			hasSlash: strings.Contains(pattern, "/"),
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

func globToRegexp(pattern string) string {
	var out strings.Builder
	for i := 0; i < len(pattern); {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				i += 2
				if i < len(pattern) && pattern[i] == '/' {
					out.WriteString("(?:.*/)?")
					i++
				} else {
					out.WriteString(".*")
				}
			} else {
				out.WriteString("[^/]*")
				i++
			}
		case '?':
			out.WriteString("[^/]")
			i++
		case '\\':
			if i+1 < len(pattern) {
				out.WriteString(regexp.QuoteMeta(pattern[i+1 : i+2]))
				i += 2
			} else {
				out.WriteString("\\\\")
				i++
			}
		default:
			out.WriteString(regexp.QuoteMeta(pattern[i : i+1]))
			i++
		}
	}
	return out.String()
}
