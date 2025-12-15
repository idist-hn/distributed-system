package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/p2p-filesharing/distributed-system/pkg/protocol"
	"github.com/p2p-filesharing/distributed-system/services/tracker/internal/models"
	"github.com/p2p-filesharing/distributed-system/services/tracker/internal/storage"
)

// getRealIP extracts the real client IP from the request
func getRealIP(r *http.Request) string {
	// Check X-Forwarded-For header (set by proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if colonIdx := strings.LastIndex(ip, ":"); colonIdx != -1 {
		ip = ip[:colonIdx]
	}
	return ip
}

// Handler holds dependencies for HTTP handlers
type Handler struct {
	storage storage.Storage
	wsHub   *WSHub
}

// NewHandler creates a new Handler
func NewHandler(s storage.Storage) *Handler {
	return &Handler{storage: s}
}

// SetWSHub sets the WebSocket hub for broadcasting events
func (h *Handler) SetWSHub(hub *WSHub) {
	h.wsHub = hub
}

// broadcastEvent sends an event to all WebSocket clients
func (h *Handler) broadcastEvent(eventType string, data interface{}) {
	if h.wsHub != nil {
		h.wsHub.Broadcast(WSEvent{
			Type: eventType,
			Data: data,
		})
	}
}

// RegisterPeer handles POST /api/peers/register
func (h *Handler) RegisterPeer(w http.ResponseWriter, r *http.Request) {
	var req protocol.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get real IP from request if peer sends localhost
	peerIP := req.IP
	if peerIP == "" || peerIP == "127.0.0.1" || peerIP == "localhost" {
		peerIP = getRealIP(r)
	}

	peer := &models.Peer{
		ID:       req.PeerID,
		IP:       peerIP,
		Port:     req.Port,
		Hostname: req.Hostname,
	}

	if err := h.storage.RegisterPeer(peer); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to register peer")
		return
	}

	// Broadcast peer joined event
	h.broadcastEvent(EventPeerJoined, map[string]interface{}{
		"peer_id":  peer.ID,
		"hostname": peer.Hostname,
		"ip":       peer.IP,
	})

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

	// Broadcast peer left event
	h.broadcastEvent(EventPeerLeft, map[string]string{
		"peer_id": peerID,
	})

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

	// Broadcast file added event
	h.broadcastEvent(EventFileAdded, map[string]interface{}{
		"hash":     file.Hash,
		"name":     file.Name,
		"size":     file.Size,
		"added_by": file.AddedBy,
	})

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

// SearchFiles handles GET /api/files/search?q=query
func (h *Handler) SearchFiles(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		sendError(w, http.StatusBadRequest, "Query parameter 'q' is required")
		return
	}

	files := h.storage.SearchFiles(query)
	sendJSON(w, http.StatusOK, map[string]interface{}{
		"query": query,
		"count": len(files),
		"files": files,
	})
}

// ListCategories handles GET /api/categories
func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	categories := h.storage.ListCategories()
	sendJSON(w, http.StatusOK, map[string]interface{}{
		"count":      len(categories),
		"categories": categories,
	})
}

// ListFilesByCategory handles GET /api/categories/{category}/files
func (h *Handler) ListFilesByCategory(w http.ResponseWriter, r *http.Request) {
	category := r.PathValue("category")
	if category == "" {
		sendError(w, http.StatusBadRequest, "Category is required")
		return
	}

	files := h.storage.ListFilesByCategory(category)
	sendJSON(w, http.StatusOK, map[string]interface{}{
		"category": category,
		"count":    len(files),
		"files":    files,
	})
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

// === Admin Endpoints ===

// AdminListPeers handles GET /api/admin/peers
func (h *Handler) AdminListPeers(w http.ResponseWriter, r *http.Request) {
	peers := h.storage.ListAllPeers()

	type PeerInfo struct {
		ID           string `json:"id"`
		IP           string `json:"ip"`
		Port         int    `json:"port"`
		Hostname     string `json:"hostname"`
		IsOnline     bool   `json:"is_online"`
		RegisteredAt string `json:"registered_at"`
		LastSeen     string `json:"last_seen"`
	}

	result := make([]PeerInfo, 0, len(peers))
	for _, p := range peers {
		result = append(result, PeerInfo{
			ID:           p.ID,
			IP:           p.IP,
			Port:         p.Port,
			Hostname:     p.Hostname,
			IsOnline:     p.IsOnline,
			RegisteredAt: p.RegisteredAt.Format("2006-01-02T15:04:05Z"),
			LastSeen:     p.LastSeen.Format("2006-01-02T15:04:05Z"),
		})
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"count": len(result),
		"peers": result,
	})
}

// AdminKickPeer handles DELETE /api/admin/peers/{peer_id}
func (h *Handler) AdminKickPeer(w http.ResponseWriter, r *http.Request) {
	peerID := r.PathValue("peer_id")
	if peerID == "" {
		sendError(w, http.StatusBadRequest, "Peer ID is required")
		return
	}

	if _, exists := h.storage.GetPeer(peerID); !exists {
		sendError(w, http.StatusNotFound, "Peer not found")
		return
	}

	if err := h.storage.RemovePeer(peerID); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to kick peer")
		return
	}

	sendJSON(w, http.StatusOK, map[string]string{
		"message": "Peer kicked successfully",
		"peer_id": peerID,
	})
}

// AdminDeleteFile handles DELETE /api/admin/files/{hash}
func (h *Handler) AdminDeleteFile(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	if hash == "" {
		sendError(w, http.StatusBadRequest, "File hash is required")
		return
	}

	if _, exists := h.storage.GetFile(hash); !exists {
		sendError(w, http.StatusNotFound, "File not found")
		return
	}

	if err := h.storage.DeleteFile(hash); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to delete file")
		return
	}

	sendJSON(w, http.StatusOK, map[string]string{
		"message": "File deleted successfully",
		"hash":    hash,
	})
}

// === Reputation Endpoints ===

// GetTopPeers handles GET /api/peers/top?limit=10
func (h *Handler) GetTopPeers(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := parseInt(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	peers := h.storage.GetTopPeers(limit)

	type PeerReputation struct {
		ID              string  `json:"id"`
		Hostname        string  `json:"hostname"`
		Reputation      float64 `json:"reputation"`
		BytesUploaded   int64   `json:"bytes_uploaded"`
		BytesDownloaded int64   `json:"bytes_downloaded"`
		FilesShared     int     `json:"files_shared"`
		Ratio           float64 `json:"ratio"`
	}

	result := make([]PeerReputation, 0, len(peers))
	for _, p := range peers {
		ratio := 0.0
		if p.BytesDownloaded > 0 {
			ratio = float64(p.BytesUploaded) / float64(p.BytesDownloaded)
		}
		result = append(result, PeerReputation{
			ID:              p.ID,
			Hostname:        p.Hostname,
			Reputation:      p.Reputation,
			BytesUploaded:   p.BytesUploaded,
			BytesDownloaded: p.BytesDownloaded,
			FilesShared:     p.FilesShared,
			Ratio:           ratio,
		})
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"count": len(result),
		"peers": result,
	})
}

// ReportStats handles POST /api/peers/stats - peers report their upload/download stats
func (h *Handler) ReportStats(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PeerID          string `json:"peer_id"`
		BytesUploaded   int64  `json:"bytes_uploaded"`
		BytesDownloaded int64  `json:"bytes_downloaded"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.PeerID == "" {
		sendError(w, http.StatusBadRequest, "peer_id is required")
		return
	}

	if err := h.storage.UpdatePeerStats(req.PeerID, req.BytesUploaded, req.BytesDownloaded); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to update stats")
		return
	}

	sendJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// Helper functions
func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
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
