package peerscore

import (
	"testing"
	"time"
)

func TestNewScorer(t *testing.T) {
	config := DefaultConfig()
	scorer := NewScorer(config)

	if scorer == nil {
		t.Fatal("NewScorer() returned nil")
	}
	if scorer.stats == nil {
		t.Error("stats map should be initialized")
	}
}

func TestScorer_RecordDownload(t *testing.T) {
	scorer := NewScorer(DefaultConfig())

	scorer.RecordDownload("peer1", 1024*1024, 100*time.Millisecond)
	scorer.RecordDownload("peer1", 2*1024*1024, 50*time.Millisecond)

	stats, ok := scorer.GetStats("peer1")
	if !ok {
		t.Fatal("GetStats() should return true for existing peer")
	}

	if stats.BytesDownloaded != 3*1024*1024 {
		t.Errorf("BytesDownloaded = %d, want %d", stats.BytesDownloaded, 3*1024*1024)
	}
	if stats.SuccessfulChunks != 2 {
		t.Errorf("SuccessfulChunks = %d, want 2", stats.SuccessfulChunks)
	}
}

func TestScorer_RecordUpload(t *testing.T) {
	scorer := NewScorer(DefaultConfig())

	scorer.RecordUpload("peer1", 512*1024)
	scorer.RecordUpload("peer1", 512*1024)

	stats, ok := scorer.GetStats("peer1")
	if !ok {
		t.Fatal("GetStats() should return true")
	}

	if stats.BytesUploaded != 1024*1024 {
		t.Errorf("BytesUploaded = %d, want %d", stats.BytesUploaded, 1024*1024)
	}
}

func TestScorer_RecordFailure(t *testing.T) {
	scorer := NewScorer(DefaultConfig())

	scorer.RecordDownload("peer1", 1024, 10*time.Millisecond)
	scorer.RecordFailure("peer1")
	scorer.RecordFailure("peer1")

	stats, ok := scorer.GetStats("peer1")
	if !ok {
		t.Fatal("GetStats() should return true")
	}

	if stats.FailedChunks != 2 {
		t.Errorf("FailedChunks = %d, want 2", stats.FailedChunks)
	}
	if stats.SuccessfulChunks != 1 {
		t.Errorf("SuccessfulChunks = %d, want 1", stats.SuccessfulChunks)
	}
}

func TestScorer_GetScore(t *testing.T) {
	scorer := NewScorer(DefaultConfig())

	// Record some activity
	scorer.RecordDownload("peer1", 5*1024*1024, 50*time.Millisecond)
	scorer.RecordUpload("peer1", 2*1024*1024)

	score := scorer.GetScore("peer1")

	if score.PeerID != "peer1" {
		t.Errorf("PeerID = %s, want peer1", score.PeerID)
	}
	if score.TotalScore <= 0 {
		t.Error("TotalScore should be positive for active peer")
	}
	if len(score.Components) == 0 {
		t.Error("Components should not be empty")
	}

	// Check components exist
	expectedComponents := []string{"download_speed", "upload_ratio", "reliability", "latency", "recency"}
	for _, comp := range expectedComponents {
		if _, ok := score.Components[comp]; !ok {
			t.Errorf("Missing component: %s", comp)
		}
	}
}

func TestScorer_GetScore_UnknownPeer(t *testing.T) {
	scorer := NewScorer(DefaultConfig())

	score := scorer.GetScore("unknown-peer")

	if score.TotalScore != 0 {
		t.Errorf("TotalScore = %f, want 0 for unknown peer", score.TotalScore)
	}
}

func TestScorer_GetTopPeers(t *testing.T) {
	scorer := NewScorer(DefaultConfig())

	// Create peers with different performance
	scorer.RecordDownload("peer1", 10*1024*1024, 10*time.Millisecond)
	scorer.RecordDownload("peer2", 5*1024*1024, 50*time.Millisecond)
	scorer.RecordDownload("peer3", 1*1024*1024, 100*time.Millisecond)

	topPeers := scorer.GetTopPeers(2)

	if len(topPeers) != 2 {
		t.Fatalf("GetTopPeers(2) returned %d peers, want 2", len(topPeers))
	}

	// First peer should have highest score
	if topPeers[0].TotalScore < topPeers[1].TotalScore {
		t.Error("Peers should be sorted by score descending")
	}
}

func TestScorer_GetTopPeers_MoreThanAvailable(t *testing.T) {
	scorer := NewScorer(DefaultConfig())

	scorer.RecordDownload("peer1", 1024, 10*time.Millisecond)

	topPeers := scorer.GetTopPeers(10)

	if len(topPeers) != 1 {
		t.Errorf("GetTopPeers(10) returned %d peers, want 1", len(topPeers))
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Weights should sum to approximately 1.0
	total := config.DownloadSpeedWeight + config.UploadRatioWeight +
		config.ReliabilityWeight + config.LatencyWeight + config.RecencyWeight

	if total < 0.99 || total > 1.01 {
		t.Errorf("Config weights sum to %f, should be ~1.0", total)
	}
}

