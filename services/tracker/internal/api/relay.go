package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Relay message types
const (
	RelayMsgRegister     = "relay_register"
	RelayMsgRequest      = "relay_request"
	RelayMsgResponse     = "relay_response"
	RelayMsgChunkRequest = "relay_chunk_request"
	RelayMsgChunkData    = "relay_chunk_data"
	RelayMsgError        = "relay_error"
	RelayMsgPing         = "relay_ping"
	RelayMsgPong         = "relay_pong"
)

// RelayMessage represents a message sent through the relay
type RelayMessage struct {
	Type      string          `json:"type"`
	From      string          `json:"from,omitempty"`
	To        string          `json:"to,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// RelayChunkRequest is the payload for chunk requests
type RelayChunkRequest struct {
	FileHash   string `json:"file_hash"`
	ChunkIndex int    `json:"chunk_index"`
}

// RelayChunkResponse is the payload for chunk responses
type RelayChunkResponse struct {
	FileHash   string `json:"file_hash"`
	ChunkIndex int    `json:"chunk_index"`
	Data       []byte `json:"data"`
	Hash       string `json:"hash"`
}

// RelayError is the payload for error messages
type RelayError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// RelayPeer represents a peer connected to the relay
type RelayPeer struct {
	ID       string
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *RelayHub
	LastSeen time.Time
	mu       sync.Mutex
}

// RelayHub manages peer-to-peer relay connections
type RelayHub struct {
	peers      map[string]*RelayPeer
	register   chan *RelayPeer
	unregister chan *RelayPeer
	relay      chan *RelayMessage
	mu         sync.RWMutex
}

// NewRelayHub creates a new relay hub
func NewRelayHub() *RelayHub {
	return &RelayHub{
		peers:      make(map[string]*RelayPeer),
		register:   make(chan *RelayPeer),
		unregister: make(chan *RelayPeer),
		relay:      make(chan *RelayMessage, 1024), // Increased buffer for better throughput
	}
}

// Run starts the relay hub main loop
func (h *RelayHub) Run() {
	for {
		select {
		case peer := <-h.register:
			h.mu.Lock()
			// Close existing connection if peer reconnects
			if existing, ok := h.peers[peer.ID]; ok {
				close(existing.Send)
				existing.Conn.Close()
			}
			h.peers[peer.ID] = peer
			h.mu.Unlock()
			log.Printf("[Relay] Peer registered: %s. Total: %d", peer.ID, len(h.peers))

		case peer := <-h.unregister:
			h.mu.Lock()
			if existing, ok := h.peers[peer.ID]; ok && existing == peer {
				delete(h.peers, peer.ID)
				close(peer.Send)
			}
			h.mu.Unlock()
			log.Printf("[Relay] Peer unregistered: %s. Total: %d", peer.ID, len(h.peers))

		case msg := <-h.relay:
			h.forwardMessage(msg)
		}
	}
}

// forwardMessage forwards a relay message to the target peer
func (h *RelayHub) forwardMessage(msg *RelayMessage) {
	fromShort := msg.From
	if len(fromShort) > 8 {
		fromShort = fromShort[:8]
	}
	toShort := msg.To
	if len(toShort) > 8 {
		toShort = toShort[:8]
	}

	h.mu.RLock()
	targetPeer, ok := h.peers[msg.To]
	connectedCount := len(h.peers)
	h.mu.RUnlock()

	if !ok {
		// Target peer not connected, send error back to sender
		log.Printf("[Relay] Target peer %s not connected (total peers: %d)", toShort, connectedCount)
		h.sendError(msg.From, msg.RequestID, 404, "Target peer not connected")
		return
	}

	// Parse chunk request for detailed logging (reduced frequency)
	if msg.Type == RelayMsgChunkRequest {
		var req RelayChunkRequest
		if err := json.Unmarshal(msg.Payload, &req); err == nil {
			// Only log every 100th chunk request to reduce noise
			if req.ChunkIndex%100 == 0 {
				fileHashShort := req.FileHash
				if len(fileHashShort) > 12 {
					fileHashShort = fileHashShort[:12]
				}
				log.Printf("[Relay] Chunk request: %s -> %s, file=%s, chunk=%d",
					fromShort, toShort, fileHashShort, req.ChunkIndex)
			}
		}
	} else if msg.Type == RelayMsgChunkData {
		var resp RelayChunkResponse
		if err := json.Unmarshal(msg.Payload, &resp); err == nil {
			// Only log every 100th chunk response
			if resp.ChunkIndex%100 == 0 {
				log.Printf("[Relay] Chunk response: %s -> %s, chunk=%d, size=%d bytes",
					fromShort, toShort, resp.ChunkIndex, len(resp.Data))
			}
		}
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[Relay] Failed to marshal message: %v", err)
		return
	}

	select {
	case targetPeer.Send <- data:
		if msg.Type != RelayMsgChunkRequest && msg.Type != RelayMsgChunkData {
			log.Printf("[Relay] Forwarded %s: %s -> %s", msg.Type, fromShort, toShort)
		}
	default:
		log.Printf("[Relay] Failed to forward to %s: channel full", toShort)
	}
}

// sendError sends an error message back to a peer
func (h *RelayHub) sendError(peerID, requestID string, code int, message string) {
	h.mu.RLock()
	peer, ok := h.peers[peerID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	errPayload, _ := json.Marshal(RelayError{Code: code, Message: message})
	msg := RelayMessage{
		Type:      RelayMsgError,
		RequestID: requestID,
		Payload:   errPayload,
		Timestamp: time.Now(),
	}

	data, _ := json.Marshal(msg)
	select {
	case peer.Send <- data:
	default:
	}
}

// GetConnectedPeers returns list of connected peer IDs
func (h *RelayHub) GetConnectedPeers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	peers := make([]string, 0, len(h.peers))
	for id := range h.peers {
		peers = append(peers, id)
	}
	return peers
}

// IsPeerConnected checks if a peer is connected to the relay
func (h *RelayHub) IsPeerConnected(peerID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.peers[peerID]
	return ok
}

// readPump reads messages from a relay peer
func (p *RelayPeer) readPump() {
	defer func() {
		p.Hub.unregister <- p
		p.Conn.Close()
	}()

	p.Conn.SetReadLimit(1024 * 1024) // 1MB max message
	p.Conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	p.Conn.SetPongHandler(func(string) error {
		p.Conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})

	for {
		_, data, err := p.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[Relay] Read error from %s: %v", p.ID, err)
			}
			break
		}

		var msg RelayMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("[Relay] Invalid message from %s: %v", p.ID, err)
			continue
		}

		msg.From = p.ID
		msg.Timestamp = time.Now()

		// Handle ping/pong locally
		if msg.Type == RelayMsgPing {
			p.sendPong(msg.RequestID)
			continue
		}

		// Forward to relay hub
		p.Hub.relay <- &msg
	}
}

// writePump writes messages to a relay peer
func (p *RelayPeer) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		p.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-p.Send:
			p.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				p.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := p.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			p.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := p.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// sendPong sends a pong response
func (p *RelayPeer) sendPong(requestID string) {
	msg := RelayMessage{
		Type:      RelayMsgPong,
		RequestID: requestID,
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(msg)
	select {
	case p.Send <- data:
	default:
	}
}

// ServeRelay handles relay WebSocket connections
func ServeRelay(hub *RelayHub, w http.ResponseWriter, r *http.Request) {
	peerID := r.URL.Query().Get("peer_id")
	if peerID == "" {
		http.Error(w, "peer_id required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[Relay] Upgrade error: %v", err)
		return
	}

	peer := &RelayPeer{
		ID:       peerID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Hub:      hub,
		LastSeen: time.Now(),
	}

	hub.register <- peer

	go peer.writePump()
	go peer.readPump()
}
