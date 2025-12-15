package holepunch

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestNewPuncher(t *testing.T) {
	p, err := NewPuncher("test-peer", 0) // 0 = random port
	if err != nil {
		t.Fatalf("NewPuncher() error = %v", err)
	}
	defer p.Stop()

	if p.peerID != "test-peer" {
		t.Errorf("peerID = %s, want test-peer", p.peerID)
	}
	if p.localPort == 0 {
		t.Error("localPort should not be 0 after binding")
	}
}

func TestPuncher_GetLocalPort(t *testing.T) {
	p, err := NewPuncher("test", 0)
	if err != nil {
		t.Fatalf("NewPuncher() error = %v", err)
	}
	defer p.Stop()

	port := p.GetLocalPort()
	if port <= 0 || port > 65535 {
		t.Errorf("GetLocalPort() = %d, want valid port", port)
	}
}

func TestPuncher_SetPublicAddress(t *testing.T) {
	p, err := NewPuncher("test", 0)
	if err != nil {
		t.Fatalf("NewPuncher() error = %v", err)
	}
	defer p.Stop()

	p.SetPublicAddress("1.2.3.4", 5000)
	addr := p.GetPublicAddress()

	if addr == nil {
		t.Fatal("GetPublicAddress() = nil")
	}
	if addr.IP != "1.2.3.4" {
		t.Errorf("IP = %s, want 1.2.3.4", addr.IP)
	}
	if addr.Port != 5000 {
		t.Errorf("Port = %d, want 5000", addr.Port)
	}
}

func TestPuncher_HasConnection(t *testing.T) {
	p, err := NewPuncher("test", 0)
	if err != nil {
		t.Fatalf("NewPuncher() error = %v", err)
	}
	defer p.Stop()

	if p.HasConnection("peer1") {
		t.Error("HasConnection() should return false initially")
	}

	// Simulate adding connection
	p.mu.Lock()
	p.peerConns["peer1"] = &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}
	p.mu.Unlock()

	if !p.HasConnection("peer1") {
		t.Error("HasConnection() should return true after adding")
	}
}

func TestPuncher_GetPeerAddress(t *testing.T) {
	p, err := NewPuncher("test", 0)
	if err != nil {
		t.Fatalf("NewPuncher() error = %v", err)
	}
	defer p.Stop()

	_, exists := p.GetPeerAddress("peer1")
	if exists {
		t.Error("GetPeerAddress() should return false initially")
	}

	expectedAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}
	p.mu.Lock()
	p.peerConns["peer1"] = expectedAddr
	p.mu.Unlock()

	addr, exists := p.GetPeerAddress("peer1")
	if !exists {
		t.Error("GetPeerAddress() should return true after adding")
	}
	if addr.Port != 1234 {
		t.Errorf("addr.Port = %d, want 1234", addr.Port)
	}
}

func TestPuncher_Start(t *testing.T) {
	p, err := NewPuncher("test", 0)
	if err != nil {
		t.Fatalf("NewPuncher() error = %v", err)
	}

	p.Start()
	time.Sleep(50 * time.Millisecond) // Let goroutine start
	p.Stop()
}

func TestPuncher_SendTo_NoConnection(t *testing.T) {
	p, err := NewPuncher("test", 0)
	if err != nil {
		t.Fatalf("NewPuncher() error = %v", err)
	}
	defer p.Stop()

	err = p.SendTo("nonexistent", []byte("test"))
	if err == nil {
		t.Error("SendTo() should return error when no connection exists")
	}
}

func TestPuncher_PunchTo_Timeout(t *testing.T) {
	p, err := NewPuncher("test", 0)
	if err != nil {
		t.Fatalf("NewPuncher() error = %v", err)
	}
	p.Start()
	defer p.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Punch to non-existent address should timeout
	err = p.PunchTo(ctx, "fake-peer", Endpoint{IP: "127.0.0.1", Port: 1})
	if err == nil {
		t.Error("PunchTo() should return error on timeout")
	}
}

func TestNewCoordinator(t *testing.T) {
	c, err := NewCoordinator(0)
	if err != nil {
		t.Fatalf("NewCoordinator() error = %v", err)
	}
	defer c.Stop()

	if c.conn == nil {
		t.Error("conn should not be nil")
	}
}

func TestCoordinator_GetEndpoint(t *testing.T) {
	c, err := NewCoordinator(0)
	if err != nil {
		t.Fatalf("NewCoordinator() error = %v", err)
	}
	defer c.Stop()

	_, exists := c.GetEndpoint("peer1")
	if exists {
		t.Error("GetEndpoint() should return false initially")
	}

	c.mu.Lock()
	c.endpoints["peer1"] = &PeerEndpoint{
		PeerID:     "peer1",
		PublicIP:   "1.2.3.4",
		PublicPort: 5000,
		LastSeen:   time.Now(),
	}
	c.mu.Unlock()

	ep, exists := c.GetEndpoint("peer1")
	if !exists {
		t.Error("GetEndpoint() should return true after adding")
	}
	if ep.PublicIP != "1.2.3.4" {
		t.Errorf("PublicIP = %s, want 1.2.3.4", ep.PublicIP)
	}
}

func TestCoordinator_GetAllEndpoints(t *testing.T) {
	c, err := NewCoordinator(0)
	if err != nil {
		t.Fatalf("NewCoordinator() error = %v", err)
	}
	defer c.Stop()

	c.mu.Lock()
	c.endpoints["peer1"] = &PeerEndpoint{PeerID: "peer1"}
	c.endpoints["peer2"] = &PeerEndpoint{PeerID: "peer2"}
	c.mu.Unlock()

	all := c.GetAllEndpoints()
	if len(all) != 2 {
		t.Errorf("GetAllEndpoints() len = %d, want 2", len(all))
	}
}

func TestEndpoint(t *testing.T) {
	ep := Endpoint{IP: "192.168.1.1", Port: 8080}
	if ep.IP != "192.168.1.1" {
		t.Errorf("IP = %s, want 192.168.1.1", ep.IP)
	}
	if ep.Port != 8080 {
		t.Errorf("Port = %d, want 8080", ep.Port)
	}
}

func TestMessage(t *testing.T) {
	msg := Message{
		Type:      MsgPunch,
		FromPeer:  "peer1",
		ToPeer:    "peer2",
		RequestID: "req-123",
		Timestamp: 12345,
	}

	if msg.Type != MsgPunch {
		t.Errorf("Type = %s, want %s", msg.Type, MsgPunch)
	}
	if msg.FromPeer != "peer1" {
		t.Errorf("FromPeer = %s, want peer1", msg.FromPeer)
	}
}

