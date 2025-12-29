package api

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/p2p-filesharing/distributed-system/services/tracker/internal/storage"
)

const Version = "1.3.0"

// ServerConfig holds server configuration
type ServerConfig struct {
	Addr           string
	PostgresURL    string
	JWTSecret      string
	APIKeys        []string
	EnableMetrics  bool
	RateLimitRPS   float64
	RateLimitBurst int
}

// DefaultServerConfig returns default configuration
func DefaultServerConfig() ServerConfig {
	apiKeys := []string{}
	if keys := os.Getenv("API_KEYS"); keys != "" {
		apiKeys = strings.Split(keys, ",")
	}
	return ServerConfig{
		Addr:           ":8080",
		PostgresURL:    os.Getenv("POSTGRES_URL"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		APIKeys:        apiKeys,
		EnableMetrics:  true,
		RateLimitRPS:   100,
		RateLimitBurst: 200,
	}
}

// Server represents the HTTP server for the tracker
type Server struct {
	handler         *Handler
	storage         storage.Storage
	config          ServerConfig
	authMW          *AuthMiddleware
	jwtManager      *JWTManager
	rateLimiter     *RateLimiter
	endpointLimiter *EndpointRateLimiter
	metrics         *Metrics
	healthChecker   *HealthChecker
	wsHub           *WSHub
	relayHub        *RelayHub
}

// NewServer creates a new tracker server with in-memory storage
func NewServer(addr string) *Server {
	config := DefaultServerConfig()
	config.Addr = addr
	return NewServerWithConfig(config)
}

// NewServerWithConfig creates a server with custom configuration
func NewServerWithConfig(config ServerConfig) *Server {
	var store storage.Storage
	var storageType string

	// Try PostgreSQL first, fall back to memory
	if config.PostgresURL != "" {
		pgStore, err := storage.NewPostgresStorage(config.PostgresURL)
		if err != nil {
			log.Printf("[Tracker] Failed to connect to PostgreSQL: %v, using memory storage", err)
			store = storage.NewMemoryStorage()
			storageType = "memory"
		} else {
			store = pgStore
			storageType = "postgresql"
			log.Println("[Tracker] Connected to PostgreSQL")
		}
	} else {
		store = storage.NewMemoryStorage()
		storageType = "memory"
	}

	wsHub := NewWSHub()
	relayHub := NewRelayHub()
	handler := NewHandler(store)
	handler.SetWSHub(wsHub)

	// Setup JWT manager
	jwtSecret := config.JWTSecret
	if jwtSecret == "" {
		jwtSecret = "p2p-tracker-secret-key-change-in-production"
	}
	jwtManager := NewJWTManager(jwtSecret, "p2p-tracker", 24*time.Hour)

	return &Server{
		handler:         handler,
		storage:         store,
		config:          config,
		authMW:          NewAuthMiddleware(),
		jwtManager:      jwtManager,
		rateLimiter:     NewRateLimiter(),
		endpointLimiter: NewEndpointRateLimiter(),
		metrics:         NewMetrics(),
		healthChecker:   NewHealthChecker(Version, store, storageType),
		wsHub:           wsHub,
		relayHub:        relayHub,
	}
}

// NewServerWithDB creates a new tracker server with PostgreSQL storage
func NewServerWithDB(addr, postgresURL string) (*Server, error) {
	config := DefaultServerConfig()
	config.Addr = addr
	config.PostgresURL = postgresURL
	return NewServerWithConfig(config), nil
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

	// Prometheus metrics (new promhttp handler)
	mux.Handle("GET /metrics", MetricsHandler())

	// JWT Auth endpoints
	mux.HandleFunc("POST /api/auth/login", s.jwtManager.HandleLogin(s.config.APIKeys))

	// Admin endpoints
	mux.HandleFunc("GET /api/admin/peers", s.handler.AdminListPeers)
	mux.HandleFunc("DELETE /api/admin/peers/{peer_id}", s.handler.AdminKickPeer)
	mux.HandleFunc("DELETE /api/admin/files/{hash}", s.handler.AdminDeleteFile)

	// WebSocket endpoint
	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWS(s.wsHub, w, r)
	})

	// Relay WebSocket endpoint for P2P tunneling
	mux.HandleFunc("GET /relay", func(w http.ResponseWriter, r *http.Request) {
		ServeRelay(s.relayHub, w, r)
	})

	// Relay status endpoint
	mux.HandleFunc("GET /api/relay/peers", s.handleRelayPeers)

	// Magnet link endpoints
	mux.HandleFunc("GET /api/files/{hash}/magnet", s.handler.GetMagnetLink)
	mux.HandleFunc("GET /api/magnet", s.handler.ParseMagnetLink)

	// Web Dashboard
	mux.HandleFunc("GET /dashboard", s.DashboardHandler())
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
			return
		}
		http.NotFound(w, r)
	})

	return mux
}

// GetWSHub returns the WebSocket hub for broadcasting events
func (s *Server) GetWSHub() *WSHub {
	return s.wsHub
}

// GetRelayHub returns the Relay hub for P2P tunneling
func (s *Server) GetRelayHub() *RelayHub {
	return s.relayHub
}

// handleRelayPeers returns list of peers connected to relay
func (s *Server) handleRelayPeers(w http.ResponseWriter, r *http.Request) {
	peers := s.relayHub.GetConnectedPeers()
	sendJSON(w, http.StatusOK, map[string]interface{}{
		"count": len(peers),
		"peers": peers,
	})
}

// StartCleanup starts a goroutine to cleanup offline peers and update metrics
func (s *Server) StartCleanup(interval, timeout time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			// Mark peers as offline if not seen recently
			s.storage.CleanupOfflinePeers(timeout)

			// Delete peers that have been offline for more than 5 minutes
			deleteTimeout := 5 * time.Minute
			deletedPeers := s.storage.DeleteOfflinePeers(deleteTimeout)
			if deletedPeers > 0 {
				log.Printf("[Tracker] Deleted %d peers offline for more than 5 minutes", deletedPeers)
			}

			// Delete files with no active peers
			deletedFiles := s.storage.DeleteOrphanFiles()
			if deletedFiles > 0 {
				log.Printf("[Tracker] Deleted %d files with no active peers", deletedFiles)
			}

			// Update metrics gauges
			peersOnline, _, filesCount := s.storage.GetStats()
			s.metrics.UpdateGauges(int64(peersOnline), int64(filesCount))

			// Broadcast stats update to WebSocket clients
			s.broadcastStats()
		}
	}()
}

// StartStatsBroadcast starts periodic stats broadcasting to WebSocket clients
func (s *Server) StartStatsBroadcast(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			s.broadcastStats()
		}
	}()
}

// broadcastStats sends current stats to all WebSocket clients
func (s *Server) broadcastStats() {
	peersOnline, peersTotal, filesCount := s.storage.GetStats()
	relayPeers := len(s.relayHub.GetConnectedPeers())
	wsClients := s.wsHub.ClientCount()

	s.wsHub.Broadcast(WSEvent{
		Type: EventStatsUpdate,
		Data: map[string]interface{}{
			"peers_online": peersOnline,
			"peers_total":  peersTotal,
			"files_count":  filesCount,
			"relay_peers":  relayPeers,
			"ws_clients":   wsClients,
		},
	})
}

// Run starts the HTTP server
func (s *Server) Run() error {
	mux := s.SetupRoutes()

	// Start WebSocket hub
	go s.wsHub.Run()
	log.Println("[Tracker] WebSocket hub started")

	// Start Relay hub
	go s.relayHub.Run()
	log.Println("[Tracker] Relay hub started")

	// Start cleanup routine (every 60s, timeout 90s)
	s.StartCleanup(60*time.Second, 90*time.Second)

	// Start stats broadcast (every 5s)
	s.StartStatsBroadcast(5 * time.Second)
	log.Println("[Tracker] Stats broadcast started (5s interval)")

	// Apply middlewares: prometheus -> rate limiter -> auth -> handler
	// But bypass middlewares for WebSocket endpoints to preserve http.Hijacker
	var handler http.Handler = mux
	handler = s.authMW.Middleware(handler)
	handler = s.rateLimiter.Middleware(handler)
	handler = PrometheusMiddleware(handler)

	// Create a wrapper that bypasses middleware for WebSocket endpoints
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// WebSocket endpoints need direct access to bypass middleware wrapping
		if r.URL.Path == "/ws" || r.URL.Path == "/relay" {
			mux.ServeHTTP(w, r)
			return
		}
		handler.ServeHTTP(w, r)
	})

	log.Printf("[Tracker] Starting server on %s (version %s)\n", s.config.Addr, Version)
	return http.ListenAndServe(s.config.Addr, finalHandler)
}
