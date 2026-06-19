package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	countersMu sync.Mutex
	counters   = make(map[string]*Counter)
)

type Stats struct {
	Downloads map[string]uint64 `json:"downloads"`
}

type Counter struct {
	mu           sync.Mutex
	saveMu       sync.Mutex
	downloads    map[string]uint64
	statsFile    string
	autosaveOnce sync.Once
}

func NewCounter(path string) *Counter {
	if path == "" {
		return nil
	}

	countersMu.Lock()
	defer countersMu.Unlock()

	if counter, ok := counters[path]; ok {
		return counter
	}

	counter := &Counter{
		downloads: make(map[string]uint64),
		statsFile: path,
	}
	counter.LoadStats()
	counters[path] = counter
	return counter
}

func (c *Counter) LoadStats() {
	if c == nil || c.statsFile == "" {
		return
	}
	if _, err := os.Stat(filepath.Dir(c.statsFile)); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(c.statsFile), 0755); err != nil {
			log.Printf("Error creating stats directory: %v\n", err)
			return
		}
	}

	data, err := os.ReadFile(c.statsFile)
	if err == nil {
		var s Stats
		if err := json.Unmarshal(data, &s); err == nil {
			// Older format support: if s.Downloads is nil (was a uint64 before)
			if s.Downloads != nil {
				c.mu.Lock()
				defer c.mu.Unlock()
				for path, count := range s.Downloads {
					c.downloads[path] = count
				}
			}
		}
	}
}

func (c *Counter) SaveStats() {
	if c == nil || c.statsFile == "" {
		return
	}

	c.saveMu.Lock()
	defer c.saveMu.Unlock()

	s := Stats{Downloads: make(map[string]uint64)}
	c.mu.Lock()
	for key, value := range c.downloads {
		s.Downloads[key] = value
	}
	c.mu.Unlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err == nil {
		// write to temp file then rename for atomic write
		tmpFile := c.statsFile + ".tmp"
		if err := os.WriteFile(tmpFile, data, 0644); err == nil {
			if err := os.Rename(tmpFile, c.statsFile); err != nil {
				log.Printf("Error replacing stats file: %v\n", err)
			}
		} else {
			log.Printf("Error saving stats: %v\n", err)
		}
	}
}

func (c *Counter) StartAutoSave() {
	if c == nil {
		return
	}
	c.autosaveOnce.Do(func() {
		go c.startStatsSaver()
	})
}

func (c *Counter) startStatsSaver() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		c.SaveStats()
	}
}

// DownloadCounter increments a counter when a file is successfully served (HTTP 200 or 206)
func (c *Counter) DownloadCounter(next http.Handler) http.Handler {
	if c == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observer := &statusObserver{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(observer, r)

		if observer.status == http.StatusOK || observer.status == http.StatusPartialContent {
			c.Increment(r.URL.Path)
		}
	})
}

func (c *Counter) Increment(path string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.downloads[path]++
	c.mu.Unlock()
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

func (c *Counter) GetTotalDownloads() uint64 {
	var total uint64
	if c == nil {
		return total
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, value := range c.downloads {
		total += value
	}
	return total
}

func (c *Counter) ExportDownloadStats(f func(key string, count uint64)) {
	if c == nil {
		return
	}
	snapshot := make(map[string]uint64)
	c.mu.Lock()
	for key, value := range c.downloads {
		snapshot[key] = value
	}
	c.mu.Unlock()

	for key, value := range snapshot {
		f(key, value)
	}
}

func (c *Counter) DeleteStat(key string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	delete(c.downloads, key)
	c.mu.Unlock()
}

func SaveAllCounters() {
	countersMu.Lock()
	all := make([]*Counter, 0, len(counters))
	for _, counter := range counters {
		all = append(all, counter)
	}
	countersMu.Unlock()

	for _, counter := range all {
		counter.SaveStats()
	}
}
