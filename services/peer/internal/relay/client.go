package relay

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Message types
const (
	MsgChunkRequest = "relay_chunk_request"
	MsgChunkData    = "relay_chunk_data"
	MsgError        = "relay_error"
	MsgPing         = "relay_ping"
	MsgPong         = "relay_pong"
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

// ChunkRequest is the payload for chunk requests
type ChunkRequest struct {
	FileHash   string `json:"file_hash"`
	ChunkIndex int    `json:"chunk_index"`
}

// ChunkResponse is the payload for chunk responses
type ChunkResponse struct {
	FileHash   string `json:"file_hash"`
	ChunkIndex int    `json:"chunk_index"`
	Data       []byte `json:"data"`
	Hash       string `json:"hash"`
}

// ErrorPayload is the payload for error messages
type ErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Client handles relay connection to tracker
type Client struct {
	peerID       string
	trackerURL   string
	conn         *websocket.Conn
	send         chan []byte
	responses    map[string]chan *RelayMessage
	chunkHandler ChunkHandler
	mu           sync.RWMutex
	connected    bool
	done         chan struct{}
	closing      bool // true when Close() is called intentionally
	reconnectCh  chan struct{}
}

// ChunkHandler is called when a chunk request is received
type ChunkHandler func(fileHash string, chunkIndex int) ([]byte, string, error)

// NewClient creates a new relay client
func NewClient(peerID, trackerURL string) *Client {
	return &Client{
		peerID:      peerID,
		trackerURL:  trackerURL,
		send:        make(chan []byte, 256),
		responses:   make(map[string]chan *RelayMessage),
		done:        make(chan struct{}),
		reconnectCh: make(chan struct{}, 1),
	}
}

// SetChunkHandler sets the handler for incoming chunk requests
func (c *Client) SetChunkHandler(handler ChunkHandler) {
	c.chunkHandler = handler
}

// Connect establishes WebSocket connection to relay
func (c *Client) Connect() error {
	if err := c.doConnect(); err != nil {
		return err
	}

	// Start reconnect handler
	go c.reconnectLoop()

	return nil
}

// doConnect performs the actual WebSocket connection
func (c *Client) doConnect() error {
	u, err := url.Parse(c.trackerURL)
	if err != nil {
		return err
	}

	// Convert http(s) to ws(s)
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	}
	u.Path = "/relay"
	u.RawQuery = fmt.Sprintf("peer_id=%s", c.peerID)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("relay connect failed: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.mu.Unlock()

	go c.readPump()
	go c.writePump()

	log.Printf("[Relay] Connected to %s", u.String())
	return nil
}

// reconnectLoop handles automatic reconnection when connection drops
func (c *Client) reconnectLoop() {
	for {
		select {
		case <-c.reconnectCh:
			c.mu.RLock()
			closing := c.closing
			c.mu.RUnlock()

			if closing {
				return
			}

			// Exponential backoff reconnect
			backoff := 5 * time.Second
			maxBackoff := 60 * time.Second

			for attempt := 1; ; attempt++ {
				log.Printf("[Relay] Reconnecting (attempt %d)...", attempt)

				if err := c.doConnect(); err != nil {
					log.Printf("[Relay] Reconnect failed: %v", err)
					time.Sleep(backoff)
					backoff = backoff * 2
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
					continue
				}

				log.Printf("[Relay] Reconnected successfully")
				break
			}

		case <-c.done:
			return
		}
	}
}

// Close closes the relay connection permanently (no reconnect)
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closing = true
	c.connected = false

	select {
	case <-c.done:
		// already closed
	default:
		close(c.done)
	}

	if c.conn != nil {
		c.conn.Close()
	}
}

// disconnect handles temporary disconnection (triggers reconnect)
func (c *Client) disconnect() {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return
	}
	c.connected = false
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	closing := c.closing
	c.mu.Unlock()

	if !closing {
		// Trigger reconnect
		select {
		case c.reconnectCh <- struct{}{}:
		default:
		}
	}
}

// IsConnected returns whether the relay is connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// RequestChunk requests a chunk from a remote peer via relay
func (c *Client) RequestChunk(targetPeerID, fileHash string, chunkIndex int) ([]byte, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("relay not connected")
	}

	requestID := uuid.New().String()

	// Create response channel
	respChan := make(chan *RelayMessage, 1)
	c.mu.Lock()
	c.responses[requestID] = respChan
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.responses, requestID)
		c.mu.Unlock()
	}()

	// Send chunk request
	payload, _ := json.Marshal(ChunkRequest{
		FileHash:   fileHash,
		ChunkIndex: chunkIndex,
	})

	msg := RelayMessage{
		Type:      MsgChunkRequest,
		To:        targetPeerID,
		RequestID: requestID,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	data, _ := json.Marshal(msg)
	c.send <- data

	// Wait for response with timeout
	select {
	case resp := <-respChan:
		if resp.Type == MsgError {
			var errPayload ErrorPayload
			json.Unmarshal(resp.Payload, &errPayload)
			return nil, fmt.Errorf("relay error: %s", errPayload.Message)
		}

		var chunkResp ChunkResponse
		if err := json.Unmarshal(resp.Payload, &chunkResp); err != nil {
			return nil, err
		}
		return chunkResp.Data, nil

	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("relay request timeout")

	case <-c.done:
		return nil, fmt.Errorf("relay connection closed")
	}
}

// readPump reads messages from relay
func (c *Client) readPump() {
	defer c.disconnect()

	for {
		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()

		if conn == nil {
			return
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[Relay] Read error: %v (will reconnect)", err)
			} else {
				log.Printf("[Relay] Connection closed (will reconnect)")
			}
			return
		}

		var msg RelayMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("[Relay] Invalid message: %v", err)
			continue
		}

		c.handleMessage(&msg)
	}
}

// writePump writes messages to relay
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.disconnect()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				return
			}
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()
			if conn == nil {
				return
			}
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("[Relay] Write error: %v", err)
				return
			}

		case <-ticker.C:
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()
			if conn == nil {
				return
			}
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[Relay] Ping failed: %v", err)
				return
			}

		case <-c.done:
			return
		}
	}
}

// handleMessage processes incoming relay messages
func (c *Client) handleMessage(msg *RelayMessage) {
	switch msg.Type {
	case MsgChunkRequest:
		c.handleChunkRequest(msg)

	case MsgChunkData, MsgError:
		// Route to waiting request
		c.mu.RLock()
		respChan, ok := c.responses[msg.RequestID]
		c.mu.RUnlock()

		if ok {
			select {
			case respChan <- msg:
			default:
			}
		}

	case MsgPong:
		// Ignore pong messages
	}
}

// handleChunkRequest handles incoming chunk requests from other peers
func (c *Client) handleChunkRequest(msg *RelayMessage) {
	fromPeer := msg.From
	if len(fromPeer) > 8 {
		fromPeer = fromPeer[:8]
	}
	log.Printf("[Relay] Chunk request from peer %s", fromPeer)

	if c.chunkHandler == nil {
		log.Printf("[Relay] No chunk handler registered")
		c.sendError(msg.From, msg.RequestID, 500, "No chunk handler")
		return
	}

	var req ChunkRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		log.Printf("[Relay] Invalid chunk request: %v", err)
		c.sendError(msg.From, msg.RequestID, 400, "Invalid request")
		return
	}

	fileHashShort := req.FileHash
	if len(fileHashShort) > 12 {
		fileHashShort = fileHashShort[:12]
	}
	log.Printf("[Relay] Request: file=%s chunk=%d from=%s", fileHashShort, req.ChunkIndex, fromPeer)

	// Get chunk data
	data, hash, err := c.chunkHandler(req.FileHash, req.ChunkIndex)
	if err != nil {
		log.Printf("[Relay] Failed to get chunk: %v", err)
		c.sendError(msg.From, msg.RequestID, 404, err.Error())
		return
	}

	log.Printf("[Relay] Sending chunk %d (%d bytes) to peer %s", req.ChunkIndex, len(data), fromPeer)

	// Send chunk response
	payload, _ := json.Marshal(ChunkResponse{
		FileHash:   req.FileHash,
		ChunkIndex: req.ChunkIndex,
		Data:       data,
		Hash:       hash,
	})

	resp := RelayMessage{
		Type:      MsgChunkData,
		To:        msg.From,
		RequestID: msg.RequestID,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	respData, _ := json.Marshal(resp)
	c.send <- respData
}

// sendError sends an error response
func (c *Client) sendError(toPeer, requestID string, code int, message string) {
	payload, _ := json.Marshal(ErrorPayload{Code: code, Message: message})
	msg := RelayMessage{
		Type:      MsgError,
		To:        toPeer,
		RequestID: requestID,
		Payload:   payload,
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(msg)
	c.send <- data
}
