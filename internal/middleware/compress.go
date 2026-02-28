package middleware

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

var (
	gzipPool  = sync.Pool{New: func() interface{} { return gzip.NewWriter(io.Discard) }}
	flatePool = sync.Pool{New: func() interface{} { f, _ := flate.NewWriter(io.Discard, flate.DefaultCompression); return f }}
	brPool    = sync.Pool{New: func() interface{} { return brotli.NewWriter(io.Discard) }}
	zstdPool  = sync.Pool{New: func() interface{} { z, _ := zstd.NewWriter(io.Discard); return z }}
)

type compressResponseWriter struct {
	http.ResponseWriter
	encoding    string
	w           io.Writer
	wroteHeader bool
	poolPut     func()
}

func (cw *compressResponseWriter) initWriter() {
	if cw.w != nil {
		return
	}

	contentType := cw.Header().Get("Content-Type")
	contentType = strings.ToLower(contentType)

	// Determine if the content type is compressible using a strict whitelist approach.
	// `http.ServeFile` already uses the standard `mime` package to detect `Content-Type`.
	// If it's an unknown type (e.g., application/octet-stream) or missing, it will bypass.
	isCompressible := false
	compressibleTypes := []string{
		"text/",
		"application/json",
		"application/javascript",
		"application/x-javascript",
		"application/xml",
		"application/xhtml+xml",
		"application/rss+xml",
		"application/atom+xml",
		"image/svg+xml",
	}

	for _, ct := range compressibleTypes {
		if strings.HasPrefix(contentType, ct) {
			isCompressible = true
			break
		}
	}

	// Only compress types that firmly benefit from it
	if !isCompressible {
		cw.w = cw.ResponseWriter // fallback to uncompressed for binaries/media/unknown
		return
	}

	cw.Header().Set("Content-Encoding", cw.encoding)
	cw.Header().Add("Vary", "Accept-Encoding")
	cw.Header().Del("Content-Length")

	switch cw.encoding {
	case "zstd":
		z := zstdPool.Get().(*zstd.Encoder)
		z.Reset(cw.ResponseWriter)
		cw.poolPut = func() { z.Close(); zstdPool.Put(z) }
		cw.w = z
	case "br":
		br := brPool.Get().(*brotli.Writer)
		br.Reset(cw.ResponseWriter)
		cw.poolPut = func() { br.Close(); brPool.Put(br) }
		cw.w = br
	case "gzip":
		gz := gzipPool.Get().(*gzip.Writer)
		gz.Reset(cw.ResponseWriter)
		cw.poolPut = func() { gz.Close(); gzipPool.Put(gz) }
		cw.w = gz
	case "deflate":
		fl := flatePool.Get().(*flate.Writer)
		fl.Reset(cw.ResponseWriter)
		cw.poolPut = func() { fl.Close(); flatePool.Put(fl) }
		cw.w = fl
	default:
		cw.w = cw.ResponseWriter
	}
}

func (cw *compressResponseWriter) WriteHeader(code int) {
	if !cw.wroteHeader {
		cw.initWriter()
		cw.wroteHeader = true
		cw.ResponseWriter.WriteHeader(code)
	}
}

func (cw *compressResponseWriter) Write(b []byte) (int, error) {
	if !cw.wroteHeader {
		cw.WriteHeader(http.StatusOK)
	}
	return cw.w.Write(b)
}

func (cw *compressResponseWriter) Close() {
	if cw.poolPut != nil {
		cw.poolPut()
	}
}

func Compress(algos []string, next http.Handler) http.Handler {
	allowed := make(map[string]bool)
	for _, a := range algos {
		allowed[strings.ToLower(a)] = true
	}

	if allowed["none"] || len(allowed) == 0 {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept-Encoding")

		// Skip compression for range requests
		if r.Header.Get("Range") != "" {
			next.ServeHTTP(w, r)
			return
		}

		var encoding string
		if allowed["zstd"] && strings.Contains(accept, "zstd") {
			encoding = "zstd"
		} else if (allowed["br"] || allowed["brotli"]) && strings.Contains(accept, "br") {
			encoding = "br"
		} else if allowed["gzip"] && strings.Contains(accept, "gzip") {
			encoding = "gzip"
		} else if allowed["deflate"] && strings.Contains(accept, "deflate") {
			encoding = "deflate"
		}

		if encoding == "" {
			next.ServeHTTP(w, r)
			return
		}

		cw := &compressResponseWriter{ResponseWriter: w, encoding: encoding}
		defer cw.Close()

		next.ServeHTTP(cw, r)
	})
}
