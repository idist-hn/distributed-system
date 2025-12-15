package storage

import (
	"sync"
	"time"

	"github.com/p2p-filesharing/distributed-system/pkg/protocol"
	"github.com/p2p-filesharing/distributed-system/services/tracker/internal/models"
)

// MemoryStorage is an in-memory implementation of the storage
type MemoryStorage struct {
	mu        sync.RWMutex
	peers     map[string]*models.Peer      // peerID -> Peer
	files     map[string]*models.File      // fileHash -> File
	filePeers map[string][]models.FilePeer // fileHash -> []FilePeer
}

// NewMemoryStorage creates a new in-memory storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		peers:     make(map[string]*models.Peer),
		files:     make(map[string]*models.File),
		filePeers: make(map[string][]models.FilePeer),
	}
}

// === Peer Operations ===

// RegisterPeer adds or updates a peer
func (s *MemoryStorage) RegisterPeer(peer *models.Peer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	peer.RegisteredAt = time.Now()
	peer.LastSeen = time.Now()
	peer.IsOnline = true
	s.peers[peer.ID] = peer
	return nil
}

// GetPeer retrieves a peer by ID
func (s *MemoryStorage) GetPeer(peerID string) (*models.Peer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	peer, exists := s.peers[peerID]
	return peer, exists
}

// UpdatePeerHeartbeat updates the last seen time
func (s *MemoryStorage) UpdatePeerHeartbeat(peerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if peer, exists := s.peers[peerID]; exists {
		peer.LastSeen = time.Now()
		peer.IsOnline = true
	}
	return nil
}

// RemovePeer removes a peer from the registry
func (s *MemoryStorage) RemovePeer(peerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.peers, peerID)

	// Also remove from file-peer relationships
	for fileHash, fps := range s.filePeers {
		var remaining []models.FilePeer
		for _, fp := range fps {
			if fp.PeerID != peerID {
				remaining = append(remaining, fp)
			}
		}
		s.filePeers[fileHash] = remaining
	}
	return nil
}

// CleanupOfflinePeers marks peers as offline if not seen recently
func (s *MemoryStorage) CleanupOfflinePeers(timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for _, peer := range s.peers {
		if now.Sub(peer.LastSeen) > timeout {
			peer.IsOnline = false
		}
	}
}

// === File Operations ===

// AddFile adds a new file to the registry
func (s *MemoryStorage) AddFile(file *models.File) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file.AddedAt = time.Now()
	s.files[file.Hash] = file
	return nil
}

// GetFile retrieves a file by hash
func (s *MemoryStorage) GetFile(hash string) (*models.File, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	file, exists := s.files[hash]
	return file, exists
}

// ListFiles returns all files
func (s *MemoryStorage) ListFiles() []protocol.FileListItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var items []protocol.FileListItem
	for _, file := range s.files {
		seeders, leechers := s.countPeers(file.Hash)
		items = append(items, protocol.FileListItem{
			Hash:     file.Hash,
			Name:     file.Name,
			Size:     file.Size,
			Seeders:  seeders,
			Leechers: leechers,
			AddedAt:  file.AddedAt,
		})
	}
	return items
}

// countPeers counts seeders and leechers for a file (must be called with lock)
func (s *MemoryStorage) countPeers(fileHash string) (seeders, leechers int) {
	for _, fp := range s.filePeers[fileHash] {
		peer, exists := s.peers[fp.PeerID]
		if !exists || !peer.IsOnline {
			continue
		}
		if fp.IsSeeder {
			seeders++
		} else {
			leechers++
		}
	}
	return
}

// === File-Peer Operations ===

// AddFilePeer associates a peer with a file
func (s *MemoryStorage) AddFilePeer(fp *models.FilePeer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fp.AddedAt = time.Now()
	fp.LastUpdated = time.Now()
	s.filePeers[fp.FileHash] = append(s.filePeers[fp.FileHash], *fp)
	return nil
}

// GetPeersForFile returns all peers that have a file
func (s *MemoryStorage) GetPeersForFile(fileHash string) []protocol.PeerFileInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []protocol.PeerFileInfo
	for _, fp := range s.filePeers[fileHash] {
		peer, exists := s.peers[fp.PeerID]
		if !exists || !peer.IsOnline {
			continue
		}
		result = append(result, protocol.PeerFileInfo{
			PeerInfo: protocol.PeerInfo{
				PeerID: peer.ID,
				IP:     peer.IP,
				Port:   peer.Port,
			},
			ChunksAvailable: fp.ChunksAvailable,
			IsSeeder:        fp.IsSeeder,
		})
	}
	return result
}
