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
	io.Writer
	http.ResponseWriter
	wroteHeader bool
}

func (w *compressResponseWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		w.Header().Del("Content-Length")
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *compressResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.Writer.Write(b)
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

		// Skip compression for typical uncompressible types or range requests
		// since dynamic compression breaks range requests and Sendfile zero-copy.
		if r.Header.Get("Range") != "" {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", encoding)
		w.Header().Add("Vary", "Accept-Encoding")

		cw := &compressResponseWriter{ResponseWriter: w}

		switch encoding {
		case "zstd":
			z := zstdPool.Get().(*zstd.Encoder)
			z.Reset(w)
			defer func() { z.Close(); zstdPool.Put(z) }()
			cw.Writer = z
		case "br":
			br := brPool.Get().(*brotli.Writer)
			br.Reset(w)
			defer func() { br.Close(); brPool.Put(br) }()
			cw.Writer = br
		case "gzip":
			gz := gzipPool.Get().(*gzip.Writer)
			gz.Reset(w)
			defer func() { gz.Close(); gzipPool.Put(gz) }()
			cw.Writer = gz
		case "deflate":
			fl := flatePool.Get().(*flate.Writer)
			fl.Reset(w)
			defer func() { fl.Close(); flatePool.Put(fl) }()
			cw.Writer = fl
		}

		next.ServeHTTP(cw, r)
	})
}
