package protocol

import "time"

// MessageType defines the type of P2P message
type MessageType string

const (
	// Tracker API message types
	MsgRegister  MessageType = "REGISTER"
	MsgHeartbeat MessageType = "HEARTBEAT"
	MsgAnnounce  MessageType = "ANNOUNCE"
	MsgGetPeers  MessageType = "GET_PEERS"
	MsgListFiles MessageType = "LIST_FILES"

	// P2P message types
	MsgHandshake    MessageType = "HANDSHAKE"
	MsgBitfield     MessageType = "BITFIELD"
	MsgHave         MessageType = "HAVE"
	MsgRequestChunk MessageType = "REQUEST_CHUNK"
	MsgChunkData    MessageType = "CHUNK_DATA"
	MsgError        MessageType = "ERROR"
)

// PeerInfo represents information about a peer
type PeerInfo struct {
	PeerID   string `json:"peer_id"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Hostname string `json:"hostname,omitempty"`
}

// ChunkInfo represents metadata about a file chunk
type ChunkInfo struct {
	Index int    `json:"index"`
	Hash  string `json:"hash"`
	Size  int64  `json:"size"`
}

// FileMetadata represents metadata about a shared file
type FileMetadata struct {
	Name       string      `json:"name"`
	Size       int64       `json:"size"`
	Hash       string      `json:"hash"`
	ChunkSize  int64       `json:"chunk_size"`
	Chunks     []ChunkInfo `json:"chunks"`
	MerkleRoot string      `json:"merkle_root,omitempty"`
}

// === Tracker API Messages ===

// RegisterRequest is sent by peer to register with tracker
type RegisterRequest struct {
	PeerID   string `json:"peer_id"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Hostname string `json:"hostname,omitempty"`
}

// RegisterResponse is returned by tracker after registration
type RegisterResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	SessionToken string `json:"session_token,omitempty"`
}

// HeartbeatRequest is sent periodically by peer to tracker
type HeartbeatRequest struct {
	PeerID      string   `json:"peer_id"`
	FilesHashes []string `json:"files_hashes"`
}

// HeartbeatResponse is returned by tracker
type HeartbeatResponse struct {
	Success           bool `json:"success"`
	NextHeartbeatSecs int  `json:"next_heartbeat_in"`
}

// AnnounceRequest is sent when peer wants to share a file
type AnnounceRequest struct {
	PeerID string       `json:"peer_id"`
	File   FileMetadata `json:"file"`
}

// AnnounceResponse is returned by tracker
type AnnounceResponse struct {
	Success bool   `json:"success"`
	FileID  string `json:"file_id,omitempty"`
	Message string `json:"message,omitempty"`
}

// PeerFileInfo represents a peer with file availability info
type PeerFileInfo struct {
	PeerInfo
	ChunksAvailable []int `json:"chunks_available"`
	IsSeeder        bool  `json:"is_seeder"`
}

// GetPeersResponse is returned when requesting peers for a file
type GetPeersResponse struct {
	FileHash   string         `json:"file_hash"`
	FileName   string         `json:"file_name"`
	FileSize   int64          `json:"file_size"`
	ChunkCount int            `json:"chunk_count"`
	ChunkSize  int64          `json:"chunk_size"`
	Chunks     []ChunkInfo    `json:"chunks"`
	Peers      []PeerFileInfo `json:"peers"`
}

// FileListItem represents a file in the list
type FileListItem struct {
	Hash     string    `json:"hash"`
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	Seeders  int       `json:"seeders"`
	Leechers int       `json:"leechers"`
	AddedAt  time.Time `json:"added_at"`
}

// ListFilesResponse is returned when listing available files
type ListFilesResponse struct {
	Files []FileListItem `json:"files"`
}

// === P2P Messages ===

// HandshakeMessage is exchanged when two peers connect
type HandshakeMessage struct {
	Type    MessageType `json:"type"`
	PeerID  string      `json:"peer_id"`
	Version string      `json:"version"`
}

// BitfieldMessage announces which chunks a peer has
type BitfieldMessage struct {
	Type     MessageType `json:"type"`
	FileHash string      `json:"file_hash"`
	Bitfield []bool      `json:"bitfield"` // true = has chunk
}

// HaveMessage announces a newly acquired chunk
type HaveMessage struct {
	Type       MessageType `json:"type"`
	FileHash   string      `json:"file_hash"`
	ChunkIndex int         `json:"chunk_index"`
}

// RequestChunkMessage requests a specific chunk
type RequestChunkMessage struct {
	Type       MessageType `json:"type"`
	FileHash   string      `json:"file_hash"`
	ChunkIndex int         `json:"chunk_index"`
}

// ChunkDataMessage contains the actual chunk data
type ChunkDataMessage struct {
	Type       MessageType `json:"type"`
	FileHash   string      `json:"file_hash"`
	ChunkIndex int         `json:"chunk_index"`
	ChunkHash  string      `json:"chunk_hash"`
	Data       []byte      `json:"data"`
}

// ErrorMessage is sent when an error occurs
type ErrorMessage struct {
	Type    MessageType `json:"type"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
}

// Error codes
const (
	ErrPeerNotFound      = 1001
	ErrFileNotFound      = 1002
	ErrChunkNotAvailable = 1003
	ErrHashMismatch      = 1004
	ErrConnectionRefused = 1005
	ErrInvalidMessage    = 1006
)
