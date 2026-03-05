package middleware

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sfsd/internal/config"
	"time"
)

type responseWriterObserver struct {
	http.ResponseWriter
	status      int
	written     int64
	wroteHeader bool
}

func (w *responseWriterObserver) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status = code
		w.wroteHeader = true
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *responseWriterObserver) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	w.written += int64(n)
	return n, err
}

func Logger(cfg *config.ServerInstance, next http.Handler) http.Handler {
	var accessLogger *log.Logger
	var errorLogger *log.Logger

	if cfg.Logging.AccessLog != "" {
		f, err := os.OpenFile(cfg.Logging.AccessLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			accessLogger = log.New(f, "", log.LstdFlags)
		} else {
			log.Printf("Failed to open access log: %v", err)
		}
	}
	if accessLogger == nil {
		accessLogger = log.New(os.Stdout, "[ACCESS] ", log.LstdFlags)
	}

	if cfg.Logging.ErrorLog != "" {
		f, err := os.OpenFile(cfg.Logging.ErrorLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			errorLogger = log.New(f, "", log.LstdFlags)
		} else {
			log.Printf("Failed to open error log: %v", err)
		}
	}
	if errorLogger == nil {
		errorLogger = log.New(os.Stderr, "[ERROR] ", log.LstdFlags)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ip := GetRealIP(r)

		observer := &responseWriterObserver{
			ResponseWriter: w,
			status:         http.StatusOK, // Default to 200 if nothing is written
		}

		next.ServeHTTP(observer, r)

		duration := time.Since(start)

		if cfg.Logging.Format == "json" {
			logEntry := map[string]interface{}{
				"timestamp": start.Format(time.RFC3339),
				"ip":        ip,
				"method":    r.Method,
				"path":      r.URL.Path,
				"proto":     r.Proto,
				"status":    observer.status,
				"bytes":     observer.written,
				"duration":  duration.String(),
			}
			jsonBytes, err := json.Marshal(logEntry)
			if err == nil {
				if observer.status >= 400 {
					errorLogger.Println(string(jsonBytes))
				} else {
					accessLogger.Println(string(jsonBytes))
				}
			}
		} else {
			msg := fmt.Sprintf("%s - %s %s %s - %d %d bytes - %v",
				ip, r.Method, r.URL.Path, r.Proto, observer.status, observer.written, duration)
			if observer.status >= 400 {
				errorLogger.Println(msg)
			} else {
				accessLogger.Println(msg)
			}
		}
	})
}
