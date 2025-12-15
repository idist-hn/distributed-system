package p2p

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/p2p-filesharing/distributed-system/pkg/hash"
	"github.com/p2p-filesharing/distributed-system/pkg/protocol"
)

// Client handles outgoing P2P connections to other peers
type Client struct {
	peerID  string
	timeout time.Duration
}

// NewClient creates a new P2P client
func NewClient(peerID string) *Client {
	return &Client{
		peerID:  peerID,
		timeout: 30 * time.Second,
	}
}

// PeerConnection represents a connection to a peer
type PeerConnection struct {
	conn    net.Conn
	encoder *json.Encoder
	decoder *json.Decoder
	peerID  string
}

// Connect establishes a connection to a peer
func (c *Client) Connect(ip string, port int) (*PeerConnection, error) {
	addr := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", addr, c.timeout)
	if err != nil {
		return nil, err
	}

	pc := &PeerConnection{
		conn:    conn,
		encoder: json.NewEncoder(conn),
		decoder: json.NewDecoder(conn),
	}

	// Perform handshake
	if err := pc.handshake(c.peerID); err != nil {
		conn.Close()
		return nil, err
	}

	return pc, nil
}

// handshake performs the P2P handshake
func (pc *PeerConnection) handshake(peerID string) error {
	// Send handshake
	msg := protocol.HandshakeMessage{
		Type:    protocol.MsgHandshake,
		PeerID:  peerID,
		Version: "1.0",
	}

	if err := pc.encoder.Encode(msg); err != nil {
		return err
	}

	// Receive handshake response
	var resp protocol.HandshakeMessage
	if err := pc.decoder.Decode(&resp); err != nil {
		return err
	}

	pc.peerID = resp.PeerID
	return nil
}

// RequestChunk requests a specific chunk from the peer
func (pc *PeerConnection) RequestChunk(fileHash string, chunkIndex int, expectedHash string) ([]byte, error) {
	// Send request
	req := protocol.RequestChunkMessage{
		Type:       protocol.MsgRequestChunk,
		FileHash:   fileHash,
		ChunkIndex: chunkIndex,
	}

	if err := pc.encoder.Encode(req); err != nil {
		return nil, err
	}

	// Receive response
	var baseMsg struct {
		Type protocol.MessageType `json:"type"`
	}

	if err := pc.decoder.Decode(&baseMsg); err != nil {
		return nil, err
	}

	if baseMsg.Type == protocol.MsgError {
		var errMsg protocol.ErrorMessage
		if err := pc.decoder.Decode(&errMsg); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("peer error %d: %s", errMsg.Code, errMsg.Message)
	}

	if baseMsg.Type != protocol.MsgChunkData {
		return nil, fmt.Errorf("unexpected message type: %s", baseMsg.Type)
	}

	var resp protocol.ChunkDataMessage
	if err := pc.decoder.Decode(&resp); err != nil {
		return nil, err
	}

	// Verify chunk hash
	if expectedHash != "" && !hash.Verify(resp.Data, expectedHash) {
		return nil, fmt.Errorf("chunk hash mismatch")
	}

	return resp.Data, nil
}

// Close closes the connection
func (pc *PeerConnection) Close() error {
	return pc.conn.Close()
}

// GetPeerID returns the remote peer's ID
func (pc *PeerConnection) GetPeerID() string {
	return pc.peerID
}

// SendBitfield sends our bitfield to the peer
func (pc *PeerConnection) SendBitfield(fileHash string, bitfield []bool) (*protocol.BitfieldMessage, error) {
	msg := protocol.BitfieldMessage{
		Type:     protocol.MsgBitfield,
		FileHash: fileHash,
		Bitfield: bitfield,
	}

	if err := pc.encoder.Encode(msg); err != nil {
		return nil, err
	}

	// Receive peer's bitfield
	var resp protocol.BitfieldMessage
	if err := pc.decoder.Decode(&resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SendHave notifies peer that we have a new chunk
func (pc *PeerConnection) SendHave(fileHash string, chunkIndex int) error {
	msg := protocol.HaveMessage{
		Type:       protocol.MsgHave,
		FileHash:   fileHash,
		ChunkIndex: chunkIndex,
	}

	return pc.encoder.Encode(msg)
}
