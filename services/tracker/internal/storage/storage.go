package storage

import (
	"strings"
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

// DeleteOfflinePeers removes peers that have been offline for more than timeout
func (s *MemoryStorage) DeleteOfflinePeers(timeout time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var toDelete []string
	for id, peer := range s.peers {
		if now.Sub(peer.LastSeen) > timeout {
			toDelete = append(toDelete, id)
		}
	}

	for _, id := range toDelete {
		delete(s.peers, id)
		// Also remove from file-peer relationships
		for fileHash, fps := range s.filePeers {
			var remaining []models.FilePeer
			for _, fp := range fps {
				if fp.PeerID != id {
					remaining = append(remaining, fp)
				}
			}
			s.filePeers[fileHash] = remaining
		}
	}

	return len(toDelete)
}

// === File Operations ===

// DeleteOrphanFiles removes files that have no peers sharing them
func (s *MemoryStorage) DeleteOrphanFiles() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	var toDelete []string
	for fileHash := range s.files {
		// Check if any peer has this file
		fps := s.filePeers[fileHash]
		hasActivePeer := false
		for _, fp := range fps {
			if peer, exists := s.peers[fp.PeerID]; exists && peer.IsOnline {
				hasActivePeer = true
				break
			}
		}
		if !hasActivePeer {
			toDelete = append(toDelete, fileHash)
		}
	}

	for _, hash := range toDelete {
		delete(s.files, hash)
		delete(s.filePeers, hash)
	}

	return len(toDelete)
}

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

// GetStats returns storage statistics
func (s *MemoryStorage) GetStats() (peersOnline, peersTotal, filesCount int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	peersTotal = len(s.peers)
	for _, p := range s.peers {
		if p.IsOnline {
			peersOnline++
		}
	}
	filesCount = len(s.files)
	return
}

// === Admin Operations ===

// ListAllPeers returns all peers
func (s *MemoryStorage) ListAllPeers() []*models.Peer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	peers := make([]*models.Peer, 0, len(s.peers))
	for _, p := range s.peers {
		peers = append(peers, p)
	}
	return peers
}

// DeleteFile removes a file and its peer associations
func (s *MemoryStorage) DeleteFile(hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.files, hash)
	delete(s.filePeers, hash)
	return nil
}

// SearchFiles searches files by name (case-insensitive)
func (s *MemoryStorage) SearchFiles(query string) []protocol.FileListItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(query)
	var items []protocol.FileListItem
	for _, file := range s.files {
		if strings.Contains(strings.ToLower(file.Name), query) {
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
	}
	return items
}

// ListFilesByCategory returns files filtered by category
func (s *MemoryStorage) ListFilesByCategory(category string) []protocol.FileListItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	category = strings.ToLower(category)
	var items []protocol.FileListItem
	for _, file := range s.files {
		if strings.ToLower(file.Category) == category {
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
	}
	return items
}

// ListCategories returns statistics for all categories
func (s *MemoryStorage) ListCategories() []CategoryStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]*CategoryStats)
	for _, file := range s.files {
		cat := file.Category
		if cat == "" {
			cat = "other"
		}
		if _, exists := stats[cat]; !exists {
			stats[cat] = &CategoryStats{Category: cat}
		}
		stats[cat].FileCount++
		stats[cat].TotalSize += file.Size
	}

	result := make([]CategoryStats, 0, len(stats))
	for _, s := range stats {
		result = append(result, *s)
	}
	return result
}

// === Reputation Operations ===

// UpdatePeerStats updates peer upload/download statistics and recalculates reputation
func (s *MemoryStorage) UpdatePeerStats(peerID string, bytesUploaded, bytesDownloaded int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	peer, exists := s.peers[peerID]
	if !exists {
		return nil
	}

	peer.BytesUploaded += bytesUploaded
	peer.BytesDownloaded += bytesDownloaded
	peer.Reputation = calculateReputation(peer)
	return nil
}

// GetTopPeers returns top peers by reputation
func (s *MemoryStorage) GetTopPeers(limit int) []*models.Peer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	peers := make([]*models.Peer, 0, len(s.peers))
	for _, p := range s.peers {
		if p.IsOnline {
			peers = append(peers, p)
		}
	}

	// Sort by reputation descending
	for i := 0; i < len(peers)-1; i++ {
		for j := i + 1; j < len(peers); j++ {
			if peers[j].Reputation > peers[i].Reputation {
				peers[i], peers[j] = peers[j], peers[i]
			}
		}
	}

	if limit > len(peers) {
		limit = len(peers)
	}
	return peers[:limit]
}

// calculateReputation calculates peer reputation score (0-100)
func calculateReputation(peer *models.Peer) float64 {
	// Base score: 50
	score := 50.0

	// Upload/download ratio bonus (max +30)
	if peer.BytesDownloaded > 0 {
		ratio := float64(peer.BytesUploaded) / float64(peer.BytesDownloaded)
		if ratio >= 1.0 {
			bonus := ratio * 10
			if bonus > 30 {
				bonus = 30
			}
			score += bonus
		} else {
			// Penalty for leechers
			score -= (1 - ratio) * 20
		}
	} else if peer.BytesUploaded > 0 {
		// Pure seeder bonus
		score += 30
	}

	// Files shared bonus (max +10)
	filesBonus := float64(peer.FilesShared) * 2
	if filesBonus > 10 {
		filesBonus = 10
	}
	score += filesBonus

	// Uptime bonus (max +10)
	// Based on how long peer has been registered
	uptimeDays := peer.LastSeen.Sub(peer.RegisteredAt).Hours() / 24
	uptimeBonus := uptimeDays * 0.5
	if uptimeBonus > 10 {
		uptimeBonus = 10
	}
	score += uptimeBonus

	// Clamp to 0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}
