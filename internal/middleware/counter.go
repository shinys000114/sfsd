package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

var (
	totalDownloads uint64
	statsFile      string
)

type Stats struct {
	Downloads uint64 `json:"downloads"`
}

func InitCounter(path string) {
	if path == "" {
		return
	}
	statsFile = path
	LoadStats()
	go startStatsSaver()
}

func LoadStats() {
	if statsFile == "" {
		return
	}
	if _, err := os.Stat(filepath.Dir(statsFile)); os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(statsFile), 0755)
	}

	data, err := os.ReadFile(statsFile)
	if err == nil {
		var s Stats
		if err := json.Unmarshal(data, &s); err == nil {
			atomic.StoreUint64(&totalDownloads, s.Downloads)
		}
	}
}

func SaveStats() {
	if statsFile == "" {
		return
	}
	s := Stats{Downloads: atomic.LoadUint64(&totalDownloads)}
	data, err := json.MarshalIndent(s, "", "  ")
	if err == nil {
		// write to temp file then rename for atomic write
		tmpFile := statsFile + ".tmp"
		if err := os.WriteFile(tmpFile, data, 0644); err == nil {
			os.Rename(tmpFile, statsFile)
		} else {
			log.Printf("Error saving stats: %v\n", err)
		}
	}
}

func startStatsSaver() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		SaveStats()
	}
}

// DownloadCounter increments a counter when a file is successfully served (HTTP 200 or 206)
func DownloadCounter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observer := &statusObserver{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(observer, r)

		if observer.status == http.StatusOK || observer.status == http.StatusPartialContent {
			atomic.AddUint64(&totalDownloads, 1)
		}
	})
}

// statusObserver is a simple ResponseWriter wrapper to capture the status code
type statusObserver struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusObserver) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status = code
		w.wroteHeader = true
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *statusObserver) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

func GetTotalDownloads() uint64 {
	return atomic.LoadUint64(&totalDownloads)
}
