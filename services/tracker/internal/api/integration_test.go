package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/p2p-filesharing/distributed-system/pkg/protocol"
)

// TestFullWorkflow tests the complete P2P workflow
func TestFullWorkflow(t *testing.T) {
	// Create tracker server
	server := NewServer(":0")
	mux := server.SetupRoutes()
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := ts.Client()

	// Step 1: Register Peer 1 (Seeder)
	t.Run("Register Seeder", func(t *testing.T) {
		req := protocol.RegisterRequest{
			PeerID: "seeder-1",
			IP:     "192.168.1.10",
			Port:   6881,
		}
		body, _ := json.Marshal(req)

		resp, err := client.Post(ts.URL+"/api/peers/register", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to register: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}

		var regResp protocol.RegisterResponse
		json.NewDecoder(resp.Body).Decode(&regResp)
		if !regResp.Success {
			t.Error("Registration should succeed")
		}
	})

	// Step 2: Seeder announces a file
	t.Run("Announce File", func(t *testing.T) {
		req := protocol.AnnounceRequest{
			PeerID: "seeder-1",
			File: protocol.FileMetadata{
				Name:      "movie.mp4",
				Size:      1048576, // 1MB
				Hash:      "abc123def456789",
				ChunkSize: 262144, // 256KB
				Chunks: []protocol.ChunkInfo{
					{Index: 0, Hash: "chunk0hash", Size: 262144},
					{Index: 1, Hash: "chunk1hash", Size: 262144},
					{Index: 2, Hash: "chunk2hash", Size: 262144},
					{Index: 3, Hash: "chunk3hash", Size: 262144},
				},
			},
		}
		body, _ := json.Marshal(req)

		resp, err := client.Post(ts.URL+"/api/files/announce", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to announce: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
	})

	// Step 3: Register Peer 2 (Leecher)
	t.Run("Register Leecher", func(t *testing.T) {
		req := protocol.RegisterRequest{
			PeerID: "leecher-1",
			IP:     "192.168.1.20",
			Port:   6882,
		}
		body, _ := json.Marshal(req)

		resp, err := client.Post(ts.URL+"/api/peers/register", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to register: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
	})

	// Step 4: Leecher lists available files
	t.Run("List Files", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/api/files")
		if err != nil {
			t.Fatalf("Failed to list files: %v", err)
		}
		defer resp.Body.Close()

		var listResp protocol.ListFilesResponse
		json.NewDecoder(resp.Body).Decode(&listResp)

		if len(listResp.Files) != 1 {
			t.Errorf("Expected 1 file, got %d", len(listResp.Files))
		}

		if listResp.Files[0].Name != "movie.mp4" {
			t.Errorf("Expected movie.mp4, got %s", listResp.Files[0].Name)
		}
	})

	// Step 5: Leecher gets peers for file
	t.Run("Get Peers for File", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/api/files/abc123def456789/peers")
		if err != nil {
			t.Fatalf("Failed to get peers: %v", err)
		}
		defer resp.Body.Close()

		var peersResp protocol.GetPeersResponse
		json.NewDecoder(resp.Body).Decode(&peersResp)

		if len(peersResp.Peers) != 1 {
			t.Errorf("Expected 1 peer, got %d", len(peersResp.Peers))
		}

		if peersResp.Peers[0].PeerID != "seeder-1" {
			t.Errorf("Expected seeder-1, got %s", peersResp.Peers[0].PeerID)
		}

		if !peersResp.Peers[0].IsSeeder {
			t.Error("Peer should be marked as seeder")
		}

		if peersResp.ChunkCount != 4 {
			t.Errorf("Expected 4 chunks, got %d", peersResp.ChunkCount)
		}
	})

	// Step 6: Health check
	t.Run("Health Check", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/health")
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
	})
}
