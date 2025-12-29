package relay

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRelayMessage(t *testing.T) {
	msg := RelayMessage{
		Type:      MsgChunkRequest,
		From:      "peer-1",
		To:        "peer-2",
		RequestID: "req-123",
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded RelayMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Type != MsgChunkRequest {
		t.Errorf("Expected type %s, got %s", MsgChunkRequest, decoded.Type)
	}
	if decoded.From != "peer-1" {
		t.Errorf("Expected from peer-1, got %s", decoded.From)
	}
}

func TestChunkRequest(t *testing.T) {
	req := ChunkRequest{
		FileHash:   "abc123",
		ChunkIndex: 5,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ChunkRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.FileHash != "abc123" {
		t.Errorf("Expected hash abc123, got %s", decoded.FileHash)
	}
	if decoded.ChunkIndex != 5 {
		t.Errorf("Expected index 5, got %d", decoded.ChunkIndex)
	}
}

func TestChunkResponse(t *testing.T) {
	resp := ChunkResponse{
		FileHash:   "abc123",
		ChunkIndex: 5,
		Data:       []byte("test data"),
		Hash:       "hash123",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ChunkResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if string(decoded.Data) != "test data" {
		t.Errorf("Expected data 'test data', got %s", string(decoded.Data))
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient("peer-123", "https://tracker.example.com")
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.peerID != "peer-123" {
		t.Errorf("Expected peerID peer-123, got %s", client.peerID)
	}
}

func TestClientIsConnected(t *testing.T) {
	client := NewClient("peer-123", "https://tracker.example.com")
	// Should be false before connecting
	if client.IsConnected() {
		t.Error("Expected IsConnected to be false before Connect")
	}
}

