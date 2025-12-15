package api

import (
	"log"
	"net/http"
	"time"

	"github.com/p2p-filesharing/distributed-system/services/tracker/internal/storage"
)

const Version = "1.2.0"

// Server represents the HTTP server for the tracker
type Server struct {
	handler       *Handler
	storage       storage.Storage
	addr          string
	dbPath        string
	authMW        *AuthMiddleware
	rateLimiter   *RateLimiter
	metrics       *Metrics
	healthChecker *HealthChecker
	wsHub         *WSHub
}

// NewServer creates a new tracker server with in-memory storage
func NewServer(addr string) *Server {
	store := storage.NewMemoryStorage()
	wsHub := NewWSHub()
	handler := NewHandler(store)
	handler.SetWSHub(wsHub)
	return &Server{
		handler:       handler,
		storage:       store,
		addr:          addr,
		authMW:        NewAuthMiddleware(),
		rateLimiter:   NewRateLimiter(),
		metrics:       NewMetrics(),
		healthChecker: NewHealthChecker(Version, store, "memory"),
		wsHub:         wsHub,
	}
}

// NewServerWithDB creates a new tracker server with database storage
func NewServerWithDB(addr, dbPath string) (*Server, error) {
	store, err := storage.NewDatabaseStorage(dbPath)
	if err != nil {
		return nil, err
	}
	wsHub := NewWSHub()
	handler := NewHandler(store)
	handler.SetWSHub(wsHub)
	return &Server{
		handler:       handler,
		storage:       store,
		addr:          addr,
		dbPath:        dbPath,
		authMW:        NewAuthMiddleware(),
		rateLimiter:   NewRateLimiter(),
		metrics:       NewMetrics(),
		healthChecker: NewHealthChecker(Version, store, "postgresql"),
		wsHub:         wsHub,
	}, nil
}

// SetupRoutes configures all HTTP routes
func (s *Server) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Peer endpoints
	mux.HandleFunc("POST /api/peers/register", s.handler.RegisterPeer)
	mux.HandleFunc("POST /api/peers/heartbeat", s.handler.Heartbeat)
	mux.HandleFunc("DELETE /api/peers/{peer_id}", s.handler.LeavePeer)
	mux.HandleFunc("GET /api/peers/top", s.handler.GetTopPeers)
	mux.HandleFunc("POST /api/peers/stats", s.handler.ReportStats)

	// File endpoints
	mux.HandleFunc("POST /api/files/announce", s.handler.AnnounceFile)
	mux.HandleFunc("GET /api/files", s.handler.ListFiles)
	mux.HandleFunc("GET /api/files/search", s.handler.SearchFiles)
	mux.HandleFunc("GET /api/files/{hash}/peers", s.handler.GetFilePeers)

	// Category endpoints
	mux.HandleFunc("GET /api/categories", s.handler.ListCategories)
	mux.HandleFunc("GET /api/categories/{category}/files", s.handler.ListFilesByCategory)

	// Health check (simple for k8s probes)
	mux.HandleFunc("GET /health", s.healthChecker.SimpleHandler())

	// Detailed health check
	getPeersCount := func() int {
		online, _, _ := s.storage.GetStats()
		return online
	}
	getFilesCount := func() int {
		_, _, files := s.storage.GetStats()
		return files
	}
	mux.HandleFunc("GET /health/detailed", s.healthChecker.DetailedHandler(getPeersCount, getFilesCount))

	// Prometheus metrics
	mux.HandleFunc("GET /metrics", s.metrics.Handler())

	// Admin endpoints
	mux.HandleFunc("GET /api/admin/peers", s.handler.AdminListPeers)
	mux.HandleFunc("DELETE /api/admin/peers/{peer_id}", s.handler.AdminKickPeer)
	mux.HandleFunc("DELETE /api/admin/files/{hash}", s.handler.AdminDeleteFile)

	// WebSocket endpoint
	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWS(s.wsHub, w, r)
	})

	return mux
}

// GetWSHub returns the WebSocket hub for broadcasting events
func (s *Server) GetWSHub() *WSHub {
	return s.wsHub
}

// StartCleanup starts a goroutine to cleanup offline peers and update metrics
func (s *Server) StartCleanup(interval, timeout time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			s.storage.CleanupOfflinePeers(timeout)

			// Update metrics gauges
			peersOnline, _, filesCount := s.storage.GetStats()
			s.metrics.UpdateGauges(int64(peersOnline), int64(filesCount))

			log.Println("[Tracker] Cleaned up offline peers")
		}
	}()
}

// Run starts the HTTP server
func (s *Server) Run() error {
	mux := s.SetupRoutes()

	// Start WebSocket hub
	go s.wsHub.Run()
	log.Println("[Tracker] WebSocket hub started")

	// Start cleanup routine (every 60s, timeout 90s)
	s.StartCleanup(60*time.Second, 90*time.Second)

	// Apply middlewares: rate limiter -> auth -> handler
	var handler http.Handler = mux
	handler = s.authMW.Middleware(handler)
	handler = s.rateLimiter.Middleware(handler)

	log.Printf("[Tracker] Starting server on %s (version %s)\n", s.addr, Version)
	return http.ListenAndServe(s.addr, handler)
}
