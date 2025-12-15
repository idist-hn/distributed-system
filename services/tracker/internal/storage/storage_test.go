package storage

import (
	"testing"
	"time"

	"github.com/p2p-filesharing/distributed-system/pkg/protocol"
	"github.com/p2p-filesharing/distributed-system/services/tracker/internal/models"
)

func TestRegisterAndGetPeer(t *testing.T) {
	s := NewMemoryStorage()

	peer := &models.Peer{
		ID:   "peer-1",
		IP:   "192.168.1.1",
		Port: 6881,
	}

	if err := s.RegisterPeer(peer); err != nil {
		t.Fatalf("RegisterPeer failed: %v", err)
	}

	got, exists := s.GetPeer("peer-1")
	if !exists {
		t.Fatal("Peer not found")
	}

	if got.IP != "192.168.1.1" {
		t.Errorf("Expected IP 192.168.1.1, got %s", got.IP)
	}

	if !got.IsOnline {
		t.Error("Peer should be online")
	}
}

func TestRemovePeer(t *testing.T) {
	s := NewMemoryStorage()

	peer := &models.Peer{ID: "peer-1", IP: "192.168.1.1", Port: 6881}
	s.RegisterPeer(peer)

	s.RemovePeer("peer-1")

	_, exists := s.GetPeer("peer-1")
	if exists {
		t.Error("Peer should be removed")
	}
}

func TestAddAndGetFile(t *testing.T) {
	s := NewMemoryStorage()

	file := &models.File{
		ID:   "file-1",
		Hash: "abc123",
		Name: "test.txt",
		Size: 1024,
		Chunks: []protocol.ChunkInfo{
			{Index: 0, Hash: "chunk0", Size: 512},
			{Index: 1, Hash: "chunk1", Size: 512},
		},
	}

	if err := s.AddFile(file); err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	got, exists := s.GetFile("abc123")
	if !exists {
		t.Fatal("File not found")
	}

	if got.Name != "test.txt" {
		t.Errorf("Expected name test.txt, got %s", got.Name)
	}

	if len(got.Chunks) != 2 {
		t.Errorf("Expected 2 chunks, got %d", len(got.Chunks))
	}
}

func TestFilePeerAssociation(t *testing.T) {
	s := NewMemoryStorage()

	// Register peer
	peer := &models.Peer{ID: "peer-1", IP: "192.168.1.1", Port: 6881}
	s.RegisterPeer(peer)

	// Add file
	file := &models.File{Hash: "abc123", Name: "test.txt", Size: 1024}
	s.AddFile(file)

	// Associate peer with file
	fp := &models.FilePeer{
		FileHash:        "abc123",
		PeerID:          "peer-1",
		ChunksAvailable: []int{0, 1, 2},
		IsSeeder:        true,
	}
	s.AddFilePeer(fp)

	// Get peers for file
	peers := s.GetPeersForFile("abc123")

	if len(peers) != 1 {
		t.Fatalf("Expected 1 peer, got %d", len(peers))
	}

	if peers[0].PeerID != "peer-1" {
		t.Errorf("Expected peer-1, got %s", peers[0].PeerID)
	}

	if !peers[0].IsSeeder {
		t.Error("Peer should be seeder")
	}
}

func TestCleanupOfflinePeers(t *testing.T) {
	s := NewMemoryStorage()

	peer := &models.Peer{ID: "peer-1", IP: "192.168.1.1", Port: 6881}
	s.RegisterPeer(peer)

	// Manually set last seen to old time
	s.mu.Lock()
	s.peers["peer-1"].LastSeen = time.Now().Add(-2 * time.Minute)
	s.mu.Unlock()

	// Cleanup with 1 minute timeout
	s.CleanupOfflinePeers(1 * time.Minute)

	got, _ := s.GetPeer("peer-1")
	if got.IsOnline {
		t.Error("Peer should be offline after cleanup")
	}
}

func TestListFiles(t *testing.T) {
	s := NewMemoryStorage()

	// Add peer and files
	peer := &models.Peer{ID: "peer-1", IP: "127.0.0.1", Port: 6881}
	s.RegisterPeer(peer)

	s.AddFile(&models.File{Hash: "hash1", Name: "file1.txt", Size: 100})
	s.AddFile(&models.File{Hash: "hash2", Name: "file2.txt", Size: 200})

	files := s.ListFiles()
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}
}
