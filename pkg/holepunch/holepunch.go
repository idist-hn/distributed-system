// Package holepunch implements UDP NAT hole punching for P2P connections
package holepunch

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// Message types for hole punch protocol
const (
	MsgPunch    = "punch"
	MsgPunchAck = "punch_ack"
	MsgData     = "data"
	MsgPing     = "ping"
	MsgPong     = "pong"
)

// Message represents a UDP hole punch message
type Message struct {
	Type      string `json:"type"`
	FromPeer  string `json:"from_peer"`
	ToPeer    string `json:"to_peer,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	Data      []byte `json:"data,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// Endpoint represents a UDP endpoint (public IP:port as seen by tracker)
type Endpoint struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

// PunchResult contains the result of a hole punch attempt
type PunchResult struct {
	Success  bool
	Endpoint *net.UDPAddr
	Error    error
}

// Puncher handles UDP hole punching
type Puncher struct {
	localConn  *net.UDPConn
	peerID     string
	localPort  int
	publicAddr *Endpoint // Public address as seen by STUN/tracker
	peerConns  map[string]*net.UDPAddr
	mu         sync.RWMutex
	onMessage  func(from string, data []byte)
	stopChan   chan struct{}
	punchChan  map[string]chan PunchResult
	punchMu    sync.Mutex
}

// NewPuncher creates a new hole puncher
func NewPuncher(peerID string, localPort int) (*Puncher, error) {
	addr := &net.UDPAddr{Port: localPort}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind UDP port %d: %w", localPort, err)
	}

	// Get actual port if 0 was specified
	actualPort := conn.LocalAddr().(*net.UDPAddr).Port

	p := &Puncher{
		localConn: conn,
		peerID:    peerID,
		localPort: actualPort,
		peerConns: make(map[string]*net.UDPAddr),
		stopChan:  make(chan struct{}),
		punchChan: make(map[string]chan PunchResult),
	}

	return p, nil
}

// Start starts the UDP listener
func (p *Puncher) Start() {
	go p.readLoop()
	log.Printf("[HolePunch] Started UDP listener on port %d", p.localPort)
}

// Stop stops the puncher
func (p *Puncher) Stop() {
	close(p.stopChan)
	p.localConn.Close()
}

// GetLocalPort returns the local UDP port
func (p *Puncher) GetLocalPort() int {
	return p.localPort
}

// SetPublicAddress sets the public address (from STUN or tracker)
func (p *Puncher) SetPublicAddress(ip string, port int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.publicAddr = &Endpoint{IP: ip, Port: port}
	log.Printf("[HolePunch] Public address set: %s:%d", ip, port)
}

// GetPublicAddress returns the public address
func (p *Puncher) GetPublicAddress() *Endpoint {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.publicAddr
}

// SetMessageHandler sets the handler for incoming data messages
func (p *Puncher) SetMessageHandler(handler func(from string, data []byte)) {
	p.onMessage = handler
}

// PunchTo attempts to punch through NAT to reach a peer
func (p *Puncher) PunchTo(ctx context.Context, targetPeerID string, targetEndpoint Endpoint) error {
	addr := &net.UDPAddr{
		IP:   net.ParseIP(targetEndpoint.IP),
		Port: targetEndpoint.Port,
	}

	// Create result channel
	p.punchMu.Lock()
	resultChan := make(chan PunchResult, 1)
	p.punchChan[targetPeerID] = resultChan
	p.punchMu.Unlock()

	defer func() {
		p.punchMu.Lock()
		delete(p.punchChan, targetPeerID)
		p.punchMu.Unlock()
	}()

	// Send multiple punch packets
	msg := Message{
		Type:      MsgPunch,
		FromPeer:  p.peerID,
		ToPeer:    targetPeerID,
		Timestamp: time.Now().UnixNano(),
	}
	data, _ := json.Marshal(msg)

	// Send punch packets multiple times (NAT might drop first few)
	for i := 0; i < 5; i++ {
		_, err := p.localConn.WriteToUDP(data, addr)
		if err != nil {
			log.Printf("[HolePunch] Send punch failed: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for ACK or timeout
	select {
	case result := <-resultChan:
		if result.Success {
			p.mu.Lock()
			p.peerConns[targetPeerID] = result.Endpoint
			p.mu.Unlock()
			log.Printf("[HolePunch] Successfully punched to %s at %s", targetPeerID, result.Endpoint)
			return nil
		}
		return result.Error
	case <-time.After(5 * time.Second):
		return fmt.Errorf("hole punch timeout for peer %s", targetPeerID)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SendTo sends data to a punched peer
func (p *Puncher) SendTo(peerID string, data []byte) error {
	p.mu.RLock()
	addr, exists := p.peerConns[peerID]
	p.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no connection to peer %s", peerID)
	}

	msg := Message{
		Type:      MsgData,
		FromPeer:  p.peerID,
		Data:      data,
		Timestamp: time.Now().UnixNano(),
	}
	msgData, _ := json.Marshal(msg)

	_, err := p.localConn.WriteToUDP(msgData, addr)
	return err
}

// HasConnection checks if a punched connection exists
func (p *Puncher) HasConnection(peerID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, exists := p.peerConns[peerID]
	return exists
}

// GetPeerAddress returns the UDP address of a connected peer
func (p *Puncher) GetPeerAddress(peerID string) (*net.UDPAddr, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	addr, exists := p.peerConns[peerID]
	return addr, exists
}

// readLoop reads incoming UDP packets
func (p *Puncher) readLoop() {
	buf := make([]byte, 65536)

	for {
		select {
		case <-p.stopChan:
			return
		default:
		}

		p.localConn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, remoteAddr, err := p.localConn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			log.Printf("[HolePunch] Read error: %v", err)
			continue
		}

		var msg Message
		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			log.Printf("[HolePunch] Invalid message: %v", err)
			continue
		}

		p.handleMessage(&msg, remoteAddr)
	}
}

// handleMessage processes incoming UDP messages
func (p *Puncher) handleMessage(msg *Message, from *net.UDPAddr) {
	switch msg.Type {
	case MsgPunch:
		// Received punch from peer, send ACK back
		log.Printf("[HolePunch] Received punch from %s at %s", msg.FromPeer, from)

		// Store peer connection
		p.mu.Lock()
		p.peerConns[msg.FromPeer] = from
		p.mu.Unlock()

		// Send ACK
		ack := Message{
			Type:      MsgPunchAck,
			FromPeer:  p.peerID,
			Timestamp: time.Now().UnixNano(),
		}
		data, _ := json.Marshal(ack)
		p.localConn.WriteToUDP(data, from)

	case MsgPunchAck:
		// Received ACK, punch successful
		log.Printf("[HolePunch] Received punch ACK from %s", msg.FromPeer)

		p.punchMu.Lock()
		if ch, exists := p.punchChan[msg.FromPeer]; exists {
			ch <- PunchResult{Success: true, Endpoint: from}
		}
		p.punchMu.Unlock()

	case MsgData:
		// Received data from peer
		if p.onMessage != nil {
			p.onMessage(msg.FromPeer, msg.Data)
		}

	case MsgPing:
		// Respond with pong
		pong := Message{
			Type:      MsgPong,
			FromPeer:  p.peerID,
			RequestID: msg.RequestID,
			Timestamp: time.Now().UnixNano(),
		}
		data, _ := json.Marshal(pong)
		p.localConn.WriteToUDP(data, from)
	}
}

// DiscoverPublicAddress uses a STUN-like method to discover public address
// This requires the tracker to echo back the peer's public address
func DiscoverPublicAddress(trackerAddr string, peerID string) (*Endpoint, error) {
	addr, err := net.ResolveUDPAddr("udp", trackerAddr)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Send discovery request
	msg := Message{
		Type:      MsgPing,
		FromPeer:  peerID,
		RequestID: "discover",
		Timestamp: time.Now().UnixNano(),
	}
	data, _ := json.Marshal(msg)
	conn.Write(data)

	// Read response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("discovery timeout: %w", err)
	}

	var resp struct {
		YourIP   string `json:"your_ip"`
		YourPort int    `json:"your_port"`
	}
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		return nil, err
	}

	return &Endpoint{IP: resp.YourIP, Port: resp.YourPort}, nil
}
