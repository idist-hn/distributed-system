package api

import (
	"log"
	"net/http"
	"time"

	"github.com/p2p-filesharing/distributed-system/services/tracker/internal/storage"
)

// Server represents the HTTP server for the tracker
type Server struct {
	handler *Handler
	storage *storage.MemoryStorage
	addr    string
}

// NewServer creates a new tracker server
func NewServer(addr string) *Server {
	store := storage.NewMemoryStorage()
	return &Server{
		handler: NewHandler(store),
		storage: store,
		addr:    addr,
	}
}

// SetupRoutes configures all HTTP routes
func (s *Server) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Peer endpoints
	mux.HandleFunc("POST /api/peers/register", s.handler.RegisterPeer)
	mux.HandleFunc("POST /api/peers/heartbeat", s.handler.Heartbeat)
	mux.HandleFunc("DELETE /api/peers/{peer_id}", s.handler.LeavePeer)

	// File endpoints
	mux.HandleFunc("POST /api/files/announce", s.handler.AnnounceFile)
	mux.HandleFunc("GET /api/files", s.handler.ListFiles)
	mux.HandleFunc("GET /api/files/{hash}/peers", s.handler.GetFilePeers)

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return mux
}

// StartCleanup starts a goroutine to cleanup offline peers
func (s *Server) StartCleanup(interval, timeout time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			s.storage.CleanupOfflinePeers(timeout)
			log.Println("[Tracker] Cleaned up offline peers")
		}
	}()
}

// Run starts the HTTP server
func (s *Server) Run() error {
	mux := s.SetupRoutes()

	// Start cleanup routine (every 60s, timeout 90s)
	s.StartCleanup(60*time.Second, 90*time.Second)

	log.Printf("[Tracker] Starting server on %s\n", s.addr)
	return http.ListenAndServe(s.addr, mux)
}
