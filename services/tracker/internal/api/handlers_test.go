package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/p2p-filesharing/distributed-system/pkg/protocol"
	"github.com/p2p-filesharing/distributed-system/services/tracker/internal/storage"
)

func setupTestHandler() *Handler {
	store := storage.NewMemoryStorage()
	return NewHandler(store)
}

func TestRegisterPeer(t *testing.T) {
	h := setupTestHandler()

	req := protocol.RegisterRequest{
		PeerID: "test-peer-1",
		IP:     "127.0.0.1",
		Port:   6881,
	}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/api/peers/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.RegisterPeer(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp protocol.RegisterResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if !resp.Success {
		t.Error("Expected success to be true")
	}
}

func TestHeartbeat(t *testing.T) {
	h := setupTestHandler()

	// First register a peer
	regReq := protocol.RegisterRequest{PeerID: "test-peer-1", IP: "127.0.0.1", Port: 6881}
	body, _ := json.Marshal(regReq)
	r := httptest.NewRequest(http.MethodPost, "/api/peers/register", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.RegisterPeer(w, r)

	// Now send heartbeat
	hbReq := protocol.HeartbeatRequest{PeerID: "test-peer-1", FilesHashes: []string{}}
	body, _ = json.Marshal(hbReq)
	r = httptest.NewRequest(http.MethodPost, "/api/peers/heartbeat", bytes.NewReader(body))
	w = httptest.NewRecorder()

	h.Heartbeat(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp protocol.HeartbeatResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if !resp.Success {
		t.Error("Expected success to be true")
	}
	if resp.NextHeartbeatSecs != 30 {
		t.Errorf("Expected next heartbeat 30, got %d", resp.NextHeartbeatSecs)
	}
}

func TestAnnounceFile(t *testing.T) {
	h := setupTestHandler()

	// First register a peer
	regReq := protocol.RegisterRequest{PeerID: "test-peer-1", IP: "127.0.0.1", Port: 6881}
	body, _ := json.Marshal(regReq)
	r := httptest.NewRequest(http.MethodPost, "/api/peers/register", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.RegisterPeer(w, r)

	// Announce a file
	announceReq := protocol.AnnounceRequest{
		PeerID: "test-peer-1",
		File: protocol.FileMetadata{
			Name:      "test.txt",
			Size:      1024,
			Hash:      "abc123def456",
			ChunkSize: 256,
			Chunks: []protocol.ChunkInfo{
				{Index: 0, Hash: "chunk0hash", Size: 256},
				{Index: 1, Hash: "chunk1hash", Size: 256},
			},
		},
	}

	body, _ = json.Marshal(announceReq)
	r = httptest.NewRequest(http.MethodPost, "/api/files/announce", bytes.NewReader(body))
	w = httptest.NewRecorder()

	h.AnnounceFile(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp protocol.AnnounceResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if !resp.Success {
		t.Error("Expected success to be true")
	}
	if resp.FileID != "abc123def456" {
		t.Errorf("Expected FileID abc123def456, got %s", resp.FileID)
	}
}

func TestListFiles(t *testing.T) {
	h := setupTestHandler()

	// Register peer and announce file first
	regReq := protocol.RegisterRequest{PeerID: "test-peer-1", IP: "127.0.0.1", Port: 6881}
	body, _ := json.Marshal(regReq)
	r := httptest.NewRequest(http.MethodPost, "/api/peers/register", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.RegisterPeer(w, r)

	announceReq := protocol.AnnounceRequest{
		PeerID: "test-peer-1",
		File: protocol.FileMetadata{
			Name: "test.txt", Size: 1024, Hash: "abc123",
			Chunks: []protocol.ChunkInfo{{Index: 0, Hash: "h1", Size: 1024}},
		},
	}
	body, _ = json.Marshal(announceReq)
	r = httptest.NewRequest(http.MethodPost, "/api/files/announce", bytes.NewReader(body))
	w = httptest.NewRecorder()
	h.AnnounceFile(w, r)

	// List files
	r = httptest.NewRequest(http.MethodGet, "/api/files", nil)
	w = httptest.NewRecorder()
	h.ListFiles(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp protocol.ListFilesResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(resp.Files))
	}
}
