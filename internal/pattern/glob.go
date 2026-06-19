package pattern

import (
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type Glob struct {
	regex    *regexp.Regexp
	hasSlash bool
}

func CompileGlob(raw string) (*Glob, error) {
	pattern := NormalizeGlob(raw)
	regex, err := regexp.Compile("^" + GlobToRegexp(pattern) + "$")
	if err != nil {
		return nil, err
	}

	return &Glob{
		regex:    regex,
		hasSlash: strings.Contains(pattern, "/"),
	}, nil
}

func (g *Glob) MatchPath(rel string) bool {
	rel = filepath.ToSlash(filepath.Clean(rel))
	if g.hasSlash {
		return g.regex.MatchString(rel)
	}

	parts := strings.Split(rel, "/")
	for _, part := range parts {
		if g.regex.MatchString(part) {
			return true
		}
	}
	return false
}

func GlobToRegexp(pattern string) string {
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

func NormalizeGlob(raw string) string {
	pattern := filepath.ToSlash(raw)
	pattern = strings.TrimPrefix(pattern, "/")
	pattern = strings.TrimPrefix(pattern, "./")
	pattern = path.Clean(pattern)
	if pattern == "." {
		return ""
	}
	return pattern
}
