package p2p

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("peer-123")
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.peerID != "peer-123" {
		t.Errorf("Expected peerID peer-123, got %s", client.peerID)
	}
}

func TestClientTimeout(t *testing.T) {
	client := NewClient("peer-123")
	// Default timeout should be 5 seconds
	if client.timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", client.timeout)
	}
}

func TestSetTimeout(t *testing.T) {
	client := NewClient("peer-123")
	client.SetTimeout(10 * time.Second)
	if client.timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", client.timeout)
	}
}
