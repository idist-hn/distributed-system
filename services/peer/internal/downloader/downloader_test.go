package downloader

import (
	"testing"

	"github.com/p2p-filesharing/distributed-system/pkg/protocol"
)

func TestDownloadStats(t *testing.T) {
	stats := &DownloadStats{
		TotalChunks: 100,
		PeerStats:   make(map[string]*PeerDownloadStats),
	}

	if stats.TotalChunks != 100 {
		t.Errorf("Expected 100 chunks, got %d", stats.TotalChunks)
	}
}

func TestPeerDownloadStats(t *testing.T) {
	peerStats := &PeerDownloadStats{
		PeerID:           "peer-123",
		ChunksDownloaded: 10,
		Failures:         2,
		Score:            100.0,
	}

	if peerStats.ChunksDownloaded != 10 {
		t.Errorf("Expected 10 chunks, got %d", peerStats.ChunksDownloaded)
	}
}

func TestChunkTask(t *testing.T) {
	task := &ChunkTask{
		Index:   5,
		Hash:    "abc123",
		Retries: 0,
	}

	if task.Index != 5 {
		t.Errorf("Expected index 5, got %d", task.Index)
	}
}

func TestAssignPeers(t *testing.T) {
	peers := []protocol.PeerFileInfo{
		{PeerInfo: protocol.PeerInfo{PeerID: "peer-1"}},
		{PeerInfo: protocol.PeerInfo{PeerID: "peer-2"}},
		{PeerInfo: protocol.PeerInfo{PeerID: "peer-3"}},
		{PeerInfo: protocol.PeerInfo{PeerID: "peer-4"}},
	}

	d := &Downloader{maxWorkers: 2}

	assigned0 := d.assignPeers(0, 2, peers)
	if len(assigned0) != 2 {
		t.Errorf("Expected 2 peers for worker 0, got %d", len(assigned0))
	}

	assigned1 := d.assignPeers(1, 2, peers)
	if len(assigned1) != 2 {
		t.Errorf("Expected 2 peers for worker 1, got %d", len(assigned1))
	}
}
