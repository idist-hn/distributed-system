package p2p

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/p2p-filesharing/distributed-system/pkg/chunker"
	"github.com/p2p-filesharing/distributed-system/pkg/protocol"
	"github.com/p2p-filesharing/distributed-system/services/peer/internal/storage"
)

// Server handles incoming P2P connections from other peers
type Server struct {
	port     int
	peerID   string
	storage  *storage.LocalStorage
	chunker  *chunker.Chunker
	listener net.Listener
}

// NewServer creates a new P2P server
func NewServer(port int, peerID string, store *storage.LocalStorage) *Server {
	return &Server{
		port:    port,
		peerID:  peerID,
		storage: store,
		chunker: chunker.New(chunker.DefaultChunkSize),
	}
}

// Start starts the P2P server
func (s *Server) Start() error {
	return s.StartWithRetry(10) // Try up to 10 different ports
}

// StartWithRetry tries to start server, automatically finding an available port if needed
func (s *Server) StartWithRetry(maxRetries int) error {
	originalPort := s.port

	for i := 0; i < maxRetries; i++ {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
		if err == nil {
			s.listener = listener
			if s.port != originalPort {
				log.Printf("[P2P Server] Port %d was busy, using port %d instead", originalPort, s.port)
			}
			log.Printf("[P2P Server] Listening on port %d\n", s.port)
			go s.acceptConnections()
			return nil
		}

		// Port is busy, try next port
		log.Printf("[P2P Server] Port %d is busy, trying port %d...", s.port, s.port+1)
		s.port++
	}

	return fmt.Errorf("could not find available port after %d attempts (tried %d-%d)", maxRetries, originalPort, s.port-1)
}

// GetPort returns the actual port the server is listening on
func (s *Server) GetPort() int {
	return s.port
}

// Stop stops the P2P server
func (s *Server) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// acceptConnections handles incoming connections
func (s *Server) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("[P2P Server] Accept error: %v\n", err)
			return
		}

		go s.handleConnection(conn)
	}
}

// handleConnection processes a single peer connection
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	log.Printf("[P2P Server] New connection from %s", remoteAddr)

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		var msg json.RawMessage
		if err := decoder.Decode(&msg); err != nil {
			log.Printf("[P2P Server] Read error: %v\n", err)
			return
		}

		// Parse message type
		var baseMsg struct {
			Type protocol.MessageType `json:"type"`
		}
		if err := json.Unmarshal(msg, &baseMsg); err != nil {
			s.sendError(encoder, protocol.ErrInvalidMessage, "Invalid message format")
			continue
		}

		switch baseMsg.Type {
		case protocol.MsgHandshake:
			s.handleHandshake(encoder)

		case protocol.MsgRequestChunk:
			var req protocol.RequestChunkMessage
			if err := json.Unmarshal(msg, &req); err != nil {
				s.sendError(encoder, protocol.ErrInvalidMessage, "Invalid request")
				continue
			}
			s.handleChunkRequest(encoder, &req)

		case protocol.MsgBitfield:
			var req protocol.BitfieldMessage
			if err := json.Unmarshal(msg, &req); err != nil {
				s.sendError(encoder, protocol.ErrInvalidMessage, "Invalid bitfield")
				continue
			}
			s.handleBitfield(encoder, &req)

		case protocol.MsgHave:
			var req protocol.HaveMessage
			if err := json.Unmarshal(msg, &req); err != nil {
				s.sendError(encoder, protocol.ErrInvalidMessage, "Invalid have message")
				continue
			}
			s.handleHave(encoder, &req)

		default:
			s.sendError(encoder, protocol.ErrInvalidMessage, "Unknown message type")
		}
	}
}

// handleHandshake responds to a handshake request
func (s *Server) handleHandshake(encoder *json.Encoder) {
	resp := protocol.HandshakeMessage{
		Type:    protocol.MsgHandshake,
		PeerID:  s.peerID,
		Version: "1.0",
	}
	encoder.Encode(resp)
}

// handleChunkRequest handles a request for a file chunk
func (s *Server) handleChunkRequest(encoder *json.Encoder, req *protocol.RequestChunkMessage) {
	log.Printf("[P2P Server] Chunk request: file=%s chunk=%d", req.FileHash[:min(12, len(req.FileHash))], req.ChunkIndex)

	// Find the file
	sharedFile, exists := s.storage.GetSharedFile(req.FileHash)
	if !exists {
		log.Printf("[P2P Server] File not found: %s", req.FileHash[:min(12, len(req.FileHash))])
		s.sendError(encoder, protocol.ErrFileNotFound, "File not found")
		return
	}

	// Read the chunk
	chunkData, err := s.chunker.ReadChunk(sharedFile.FilePath, req.ChunkIndex)
	if err != nil {
		log.Printf("[P2P Server] Failed to read chunk %d: %v", req.ChunkIndex, err)
		s.sendError(encoder, protocol.ErrChunkNotAvailable, "Could not read chunk")
		return
	}

	// Get chunk hash for verification
	chunkHash := ""
	if req.ChunkIndex < len(sharedFile.Metadata.Chunks) {
		chunkHash = sharedFile.Metadata.Chunks[req.ChunkIndex].Hash
	}

	// Send the chunk
	resp := protocol.ChunkDataMessage{
		Type:       protocol.MsgChunkData,
		FileHash:   req.FileHash,
		ChunkIndex: req.ChunkIndex,
		ChunkHash:  chunkHash,
		Data:       chunkData,
	}
	log.Printf("[P2P Server] Sending chunk %d (%d bytes) for file %s",
		req.ChunkIndex, len(chunkData), sharedFile.Metadata.Name)
	encoder.Encode(resp)
}

// handleBitfield handles bitfield messages (chunks a peer has)
func (s *Server) handleBitfield(encoder *json.Encoder, req *protocol.BitfieldMessage) {
	// Get our bitfield for this file
	sharedFile, exists := s.storage.GetSharedFile(req.FileHash)
	if !exists {
		// We don't have this file, send empty bitfield
		resp := protocol.BitfieldMessage{
			Type:     protocol.MsgBitfield,
			FileHash: req.FileHash,
			Bitfield: []bool{},
		}
		encoder.Encode(resp)
		return
	}

	// Build our bitfield (we're a seeder, so we have all chunks)
	bitfield := make([]bool, len(sharedFile.Metadata.Chunks))
	for i := range bitfield {
		bitfield[i] = true
	}

	resp := protocol.BitfieldMessage{
		Type:     protocol.MsgBitfield,
		FileHash: req.FileHash,
		Bitfield: bitfield,
	}
	encoder.Encode(resp)
}

// handleHave handles have messages (peer got a new chunk)
func (s *Server) handleHave(encoder *json.Encoder, req *protocol.HaveMessage) {
	// Acknowledge the have message
	// In a full implementation, we'd track which chunks each peer has
	log.Printf("[P2P Server] Peer has chunk %d of file %s", req.ChunkIndex, req.FileHash[:8])
}

// sendError sends an error message
func (s *Server) sendError(encoder *json.Encoder, code int, message string) {
	resp := protocol.ErrorMessage{
		Type:    protocol.MsgError,
		Code:    code,
		Message: message,
	}
	encoder.Encode(resp)
}
