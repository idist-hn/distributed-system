// Package peerscore implements peer scoring and selection for P2P networks
package peerscore

import (
	"sort"
	"sync"
	"time"
)

// PeerStats contains statistics about a peer's performance
type PeerStats struct {
	PeerID           string
	BytesDownloaded  int64
	BytesUploaded    int64
	SuccessfulChunks int
	FailedChunks     int
	AverageLatency   time.Duration
	LastSeen         time.Time
	ConnectionCount  int
	IsChoking        bool
	IsInterested     bool
}

// Score represents a peer's calculated score
type Score struct {
	PeerID     string
	TotalScore float64
	Components map[string]float64
}

// Scorer calculates peer scores based on various metrics
type Scorer struct {
	mu     sync.RWMutex
	stats  map[string]*PeerStats
	config ScorerConfig
}

// ScorerConfig contains scoring weights
type ScorerConfig struct {
	DownloadSpeedWeight float64 // Weight for download speed
	UploadRatioWeight   float64 // Weight for upload/download ratio
	ReliabilityWeight   float64 // Weight for success rate
	LatencyWeight       float64 // Weight for latency (lower is better)
	RecencyWeight       float64 // Weight for recent activity
}

// DefaultConfig returns default scoring configuration
func DefaultConfig() ScorerConfig {
	return ScorerConfig{
		DownloadSpeedWeight: 0.3,
		UploadRatioWeight:   0.2,
		ReliabilityWeight:   0.25,
		LatencyWeight:       0.15,
		RecencyWeight:       0.1,
	}
}

// NewScorer creates a new peer scorer
func NewScorer(config ScorerConfig) *Scorer {
	return &Scorer{
		stats:  make(map[string]*PeerStats),
		config: config,
	}
}

// UpdateStats updates statistics for a peer
func (s *Scorer) UpdateStats(peerID string, update func(*PeerStats)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stats, ok := s.stats[peerID]
	if !ok {
		stats = &PeerStats{PeerID: peerID, LastSeen: time.Now()}
		s.stats[peerID] = stats
	}
	update(stats)
	stats.LastSeen = time.Now()
}

// RecordDownload records a successful chunk download
func (s *Scorer) RecordDownload(peerID string, bytes int64, latency time.Duration) {
	s.UpdateStats(peerID, func(stats *PeerStats) {
		stats.BytesDownloaded += bytes
		stats.SuccessfulChunks++
		// Update average latency with exponential moving average
		if stats.AverageLatency == 0 {
			stats.AverageLatency = latency
		} else {
			stats.AverageLatency = (stats.AverageLatency*7 + latency*3) / 10
		}
	})
}

// RecordUpload records bytes uploaded to a peer
func (s *Scorer) RecordUpload(peerID string, bytes int64) {
	s.UpdateStats(peerID, func(stats *PeerStats) {
		stats.BytesUploaded += bytes
	})
}

// RecordFailure records a failed chunk transfer
func (s *Scorer) RecordFailure(peerID string) {
	s.UpdateStats(peerID, func(stats *PeerStats) {
		stats.FailedChunks++
	})
}

// GetScore calculates the score for a peer
func (s *Scorer) GetScore(peerID string) Score {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats, ok := s.stats[peerID]
	if !ok {
		return Score{PeerID: peerID, TotalScore: 0, Components: make(map[string]float64)}
	}

	components := make(map[string]float64)

	// Download speed score (normalized to 0-1, assuming max 10MB/s)
	downloadSpeed := float64(stats.BytesDownloaded) / (time.Since(stats.LastSeen).Seconds() + 1)
	components["download_speed"] = min(downloadSpeed/(10*1024*1024), 1.0)

	// Upload ratio score (tit-for-tat)
	if stats.BytesDownloaded > 0 {
		ratio := float64(stats.BytesUploaded) / float64(stats.BytesDownloaded)
		components["upload_ratio"] = min(ratio, 1.0)
	}

	// Reliability score
	totalChunks := stats.SuccessfulChunks + stats.FailedChunks
	if totalChunks > 0 {
		components["reliability"] = float64(stats.SuccessfulChunks) / float64(totalChunks)
	}

	// Latency score (lower is better, normalized assuming max 1s)
	if stats.AverageLatency > 0 {
		latencyScore := 1.0 - min(float64(stats.AverageLatency)/float64(time.Second), 1.0)
		components["latency"] = latencyScore
	}

	// Recency score (how recently we've seen this peer)
	recencyScore := 1.0 - min(time.Since(stats.LastSeen).Minutes()/60, 1.0)
	components["recency"] = recencyScore

	// Calculate total score
	total := components["download_speed"]*s.config.DownloadSpeedWeight +
		components["upload_ratio"]*s.config.UploadRatioWeight +
		components["reliability"]*s.config.ReliabilityWeight +
		components["latency"]*s.config.LatencyWeight +
		components["recency"]*s.config.RecencyWeight

	return Score{PeerID: peerID, TotalScore: total, Components: components}
}

// GetTopPeers returns the top N peers by score
func (s *Scorer) GetTopPeers(n int) []Score {
	s.mu.RLock()
	peerIDs := make([]string, 0, len(s.stats))
	for id := range s.stats {
		peerIDs = append(peerIDs, id)
	}
	s.mu.RUnlock()

	scores := make([]Score, 0, len(peerIDs))
	for _, id := range peerIDs {
		scores = append(scores, s.GetScore(id))
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].TotalScore > scores[j].TotalScore
	})

	if n > len(scores) {
		n = len(scores)
	}
	return scores[:n]
}

// GetStats returns stats for a peer
func (s *Scorer) GetStats(peerID string) (*PeerStats, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats, ok := s.stats[peerID]
	if !ok {
		return nil, false
	}
	// Return a copy
	copy := *stats
	return &copy, true
}

