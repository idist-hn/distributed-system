package models

import (
	"time"

	"github.com/p2p-filesharing/distributed-system/pkg/protocol"
)

// Peer represents a registered peer in the network
type Peer struct {
	ID              string    `json:"id"`
	IP              string    `json:"ip"`
	Port            int       `json:"port"`
	Hostname        string    `json:"hostname"`
	RegisteredAt    time.Time `json:"registered_at"`
	LastSeen        time.Time `json:"last_seen"`
	IsOnline        bool      `json:"is_online"`
	BytesUploaded   int64     `json:"bytes_uploaded"`
	BytesDownloaded int64     `json:"bytes_downloaded"`
	FilesShared     int       `json:"files_shared"`
	Reputation      float64   `json:"reputation"` // Calculated reputation score 0-100
}

// File represents a shared file's metadata
type File struct {
	ID        string               `json:"id"`
	Hash      string               `json:"hash"`
	Name      string               `json:"name"`
	Size      int64                `json:"size"`
	ChunkSize int64                `json:"chunk_size"`
	Chunks    []protocol.ChunkInfo `json:"chunks"`
	Category  string               `json:"category,omitempty"` // Category: video, audio, document, image, software, other
	Tags      []string             `json:"tags,omitempty"`     // User-defined tags
	AddedAt   time.Time            `json:"added_at"`
	AddedBy   string               `json:"added_by"` // PeerID
}

// FilePeer represents the relationship between a file and a peer
type FilePeer struct {
	FileHash        string    `json:"file_hash"`
	PeerID          string    `json:"peer_id"`
	ChunksAvailable []int     `json:"chunks_available"`
	IsSeeder        bool      `json:"is_seeder"`
	AddedAt         time.Time `json:"added_at"`
	LastUpdated     time.Time `json:"last_updated"`
}
