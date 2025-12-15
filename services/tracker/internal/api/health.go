package api

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

// HealthResponse represents the detailed health check response
type HealthResponse struct {
	Status    string         `json:"status"`
	Timestamp string         `json:"timestamp"`
	Uptime    string         `json:"uptime"`
	Version   string         `json:"version"`
	Database  DatabaseHealth `json:"database"`
	Memory    MemoryHealth   `json:"memory"`
	Stats     TrackerStats   `json:"stats"`
}

// DatabaseHealth represents database health status
type DatabaseHealth struct {
	Status      string `json:"status"`
	Type        string `json:"type"`
	Latency     string `json:"latency,omitempty"`
	Error       string `json:"error,omitempty"`
}

// MemoryHealth represents memory usage
type MemoryHealth struct {
	AllocMB      float64 `json:"alloc_mb"`
	TotalAllocMB float64 `json:"total_alloc_mb"`
	SysMB        float64 `json:"sys_mb"`
	NumGC        uint32  `json:"num_gc"`
	Goroutines   int     `json:"goroutines"`
}

// TrackerStats represents tracker statistics
type TrackerStats struct {
	PeersOnline int `json:"peers_online"`
	FilesCount  int `json:"files_count"`
}

// HealthChecker provides health check functionality
type HealthChecker struct {
	startTime   time.Time
	version     string
	storage     interface{}
	dbType      string
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(version string, storage interface{}, dbType string) *HealthChecker {
	return &HealthChecker{
		startTime: time.Now(),
		version:   version,
		storage:   storage,
		dbType:    dbType,
	}
}

// SimpleHandler returns a simple health check (for k8s probes)
func (h *HealthChecker) SimpleHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}

// DetailedHandler returns detailed health information
func (h *HealthChecker) DetailedHandler(getPeersCount, getFilesCount func() int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		uptime := time.Since(h.startTime)

		// Check database
		dbHealth := DatabaseHealth{
			Status: "healthy",
			Type:   h.dbType,
		}

		// Try to ping database if it's a DatabaseStorage
		if pinger, ok := h.storage.(interface{ Ping() error }); ok {
			start := time.Now()
			if err := pinger.Ping(); err != nil {
				dbHealth.Status = "unhealthy"
				dbHealth.Error = err.Error()
			} else {
				dbHealth.Latency = time.Since(start).String()
			}
		}

		status := "healthy"
		if dbHealth.Status != "healthy" {
			status = "degraded"
		}

		resp := HealthResponse{
			Status:    status,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Uptime:    uptime.Round(time.Second).String(),
			Version:   h.version,
			Database:  dbHealth,
			Memory: MemoryHealth{
				AllocMB:      float64(memStats.Alloc) / 1024 / 1024,
				TotalAllocMB: float64(memStats.TotalAlloc) / 1024 / 1024,
				SysMB:        float64(memStats.Sys) / 1024 / 1024,
				NumGC:        memStats.NumGC,
				Goroutines:   runtime.NumGoroutine(),
			},
			Stats: TrackerStats{
				PeersOnline: getPeersCount(),
				FilesCount:  getFilesCount(),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if status != "healthy" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(resp)
	}
}

