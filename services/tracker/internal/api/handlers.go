package api

import (
	"encoding/json"
	"net/http"

	"github.com/p2p-filesharing/distributed-system/pkg/protocol"
	"github.com/p2p-filesharing/distributed-system/services/tracker/internal/models"
	"github.com/p2p-filesharing/distributed-system/services/tracker/internal/storage"
)

// Handler holds dependencies for HTTP handlers
type Handler struct {
	storage *storage.MemoryStorage
}

// NewHandler creates a new Handler
func NewHandler(s *storage.MemoryStorage) *Handler {
	return &Handler{storage: s}
}

// RegisterPeer handles POST /api/peers/register
func (h *Handler) RegisterPeer(w http.ResponseWriter, r *http.Request) {
	var req protocol.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	peer := &models.Peer{
		ID:       req.PeerID,
		IP:       req.IP,
		Port:     req.Port,
		Hostname: req.Hostname,
	}

	if err := h.storage.RegisterPeer(peer); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to register peer")
		return
	}

	sendJSON(w, http.StatusOK, protocol.RegisterResponse{
		Success: true,
		Message: "Registered successfully",
	})
}

// Heartbeat handles POST /api/peers/heartbeat
func (h *Handler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var req protocol.HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.storage.UpdatePeerHeartbeat(req.PeerID); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to update heartbeat")
		return
	}

	sendJSON(w, http.StatusOK, protocol.HeartbeatResponse{
		Success:           true,
		NextHeartbeatSecs: 30,
	})
}

// LeavePeer handles DELETE /api/peers/{peer_id}
func (h *Handler) LeavePeer(w http.ResponseWriter, r *http.Request) {
	peerID := r.PathValue("peer_id")
	if peerID == "" {
		sendError(w, http.StatusBadRequest, "Peer ID required")
		return
	}

	if err := h.storage.RemovePeer(peerID); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to remove peer")
		return
	}

	sendJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// AnnounceFile handles POST /api/files/announce
func (h *Handler) AnnounceFile(w http.ResponseWriter, r *http.Request) {
	var req protocol.AnnounceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Add file metadata
	file := &models.File{
		ID:        req.File.Hash,
		Hash:      req.File.Hash,
		Name:      req.File.Name,
		Size:      req.File.Size,
		ChunkSize: req.File.ChunkSize,
		Chunks:    req.File.Chunks,
		AddedBy:   req.PeerID,
	}
	h.storage.AddFile(file)

	// Associate peer with file (as seeder with all chunks)
	allChunks := make([]int, len(req.File.Chunks))
	for i := range req.File.Chunks {
		allChunks[i] = i
	}

	filePeer := &models.FilePeer{
		FileHash:        req.File.Hash,
		PeerID:          req.PeerID,
		ChunksAvailable: allChunks,
		IsSeeder:        true,
	}
	h.storage.AddFilePeer(filePeer)

	sendJSON(w, http.StatusOK, protocol.AnnounceResponse{
		Success: true,
		FileID:  req.File.Hash,
	})
}

// ListFiles handles GET /api/files
func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	files := h.storage.ListFiles()
	sendJSON(w, http.StatusOK, protocol.ListFilesResponse{Files: files})
}

// GetFilePeers handles GET /api/files/{hash}/peers
func (h *Handler) GetFilePeers(w http.ResponseWriter, r *http.Request) {
	fileHash := r.PathValue("hash")
	if fileHash == "" {
		sendError(w, http.StatusBadRequest, "File hash required")
		return
	}

	file, exists := h.storage.GetFile(fileHash)
	if !exists {
		sendError(w, http.StatusNotFound, "File not found")
		return
	}

	peers := h.storage.GetPeersForFile(fileHash)

	sendJSON(w, http.StatusOK, protocol.GetPeersResponse{
		FileHash:   file.Hash,
		FileName:   file.Name,
		FileSize:   file.Size,
		ChunkCount: len(file.Chunks),
		ChunkSize:  file.ChunkSize,
		Chunks:     file.Chunks,
		Peers:      peers,
	})
}

// Helper functions
func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func sendError(w http.ResponseWriter, status int, message string) {
	sendJSON(w, status, map[string]string{"error": message})
}
