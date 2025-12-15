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
}

// ChunkHandler is called when a chunk request is received
type ChunkHandler func(fileHash string, chunkIndex int) ([]byte, string, error)

// NewClient creates a new relay client
func NewClient(peerID, trackerURL string) *Client {
	return &Client{
		peerID:     peerID,
		trackerURL: trackerURL,
		send:       make(chan []byte, 256),
		responses:  make(map[string]chan *RelayMessage),
		done:       make(chan struct{}),
	}
}

// SetChunkHandler sets the handler for incoming chunk requests
func (c *Client) SetChunkHandler(handler ChunkHandler) {
	c.chunkHandler = handler
}

// Connect establishes WebSocket connection to relay
func (c *Client) Connect() error {
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

	c.conn = conn
	c.connected = true

	go c.readPump()
	go c.writePump()

	log.Printf("[Relay] Connected to %s", u.String())
	return nil
}

// Close closes the relay connection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return
	}

	c.connected = false
	close(c.done)
	if c.conn != nil {
		c.conn.Close()
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
	defer c.Close()

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[Relay] Read error: %v", err)
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
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
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
	if c.chunkHandler == nil {
		c.sendError(msg.From, msg.RequestID, 500, "No chunk handler")
		return
	}

	var req ChunkRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		c.sendError(msg.From, msg.RequestID, 400, "Invalid request")
		return
	}

	// Get chunk data
	data, hash, err := c.chunkHandler(req.FileHash, req.ChunkIndex)
	if err != nil {
		c.sendError(msg.From, msg.RequestID, 404, err.Error())
		return
	}

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
