package storage

import (
	"time"

	"github.com/p2p-filesharing/distributed-system/pkg/protocol"
	"github.com/p2p-filesharing/distributed-system/services/tracker/internal/models"
)

// CategoryStats represents statistics for a file category
type CategoryStats struct {
	Category  string `json:"category"`
	FileCount int    `json:"file_count"`
	TotalSize int64  `json:"total_size"`
}

// Storage defines the interface for tracker storage backends
type Storage interface {
	// Peer operations
	RegisterPeer(peer *models.Peer) error
	GetPeer(peerID string) (*models.Peer, bool)
	UpdatePeerHeartbeat(peerID string) error
	RemovePeer(peerID string) error
	CleanupOfflinePeers(timeout time.Duration)
	DeleteOfflinePeers(timeout time.Duration) int // Delete peers offline for more than timeout

	// File operations
	AddFile(file *models.File) error
	GetFile(hash string) (*models.File, bool)
	ListFiles() []protocol.FileListItem
	SearchFiles(query string) []protocol.FileListItem
	ListFilesByCategory(category string) []protocol.FileListItem
	ListCategories() []CategoryStats
	DeleteOrphanFiles() int // Delete files with no peers

	// File-Peer operations
	AddFilePeer(fp *models.FilePeer) error
	GetPeersForFile(fileHash string) []protocol.PeerFileInfo

	// Stats
	GetStats() (peersOnline, peersTotal, filesCount int)

	// Admin operations
	ListAllPeers() []*models.Peer
	DeleteFile(hash string) error

	// Reputation operations
	UpdatePeerStats(peerID string, bytesUploaded, bytesDownloaded int64) error
	GetTopPeers(limit int) []*models.Peer
}

// Ensure implementations satisfy the interface
var _ Storage = (*MemoryStorage)(nil)
var _ Storage = (*DatabaseStorage)(nil)
