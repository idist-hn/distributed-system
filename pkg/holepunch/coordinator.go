package holepunch

import (
	"encoding/json"
	"log"
	"net"
	"sync"
	"time"
)

// PeerEndpoint stores a peer's UDP endpoint information
type PeerEndpoint struct {
	PeerID     string
	PublicIP   string
	PublicPort int
	LocalPort  int
	LastSeen   time.Time
}

// Coordinator manages hole punch coordination on the tracker side
type Coordinator struct {
	conn      *net.UDPConn
	port      int
	endpoints map[string]*PeerEndpoint
	mu        sync.RWMutex
	stopChan  chan struct{}
}

// NewCoordinator creates a new hole punch coordinator
func NewCoordinator(port int) (*Coordinator, error) {
	addr := &net.UDPAddr{Port: port}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	return &Coordinator{
		conn:      conn,
		port:      port,
		endpoints: make(map[string]*PeerEndpoint),
		stopChan:  make(chan struct{}),
	}, nil
}

// Start starts the coordinator
func (c *Coordinator) Start() {
	go c.readLoop()
	go c.cleanupLoop()
	log.Printf("[HolePunch Coordinator] Started on UDP port %d", c.port)
}

// Stop stops the coordinator
func (c *Coordinator) Stop() {
	close(c.stopChan)
	c.conn.Close()
}

// GetEndpoint returns a peer's UDP endpoint
func (c *Coordinator) GetEndpoint(peerID string) (*PeerEndpoint, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ep, exists := c.endpoints[peerID]
	return ep, exists
}

// GetAllEndpoints returns all registered endpoints
func (c *Coordinator) GetAllEndpoints() map[string]*PeerEndpoint {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[string]*PeerEndpoint)
	for k, v := range c.endpoints {
		result[k] = v
	}
	return result
}

// readLoop reads incoming UDP packets
func (c *Coordinator) readLoop() {
	buf := make([]byte, 4096)

	for {
		select {
		case <-c.stopChan:
			return
		default:
		}

		c.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, remoteAddr, err := c.conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			continue
		}

		var msg Message
		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			continue
		}

		c.handleMessage(&msg, remoteAddr)
	}
}

// handleMessage processes incoming messages
func (c *Coordinator) handleMessage(msg *Message, from *net.UDPAddr) {
	switch msg.Type {
	case MsgPing:
		// Peer is registering/discovering their public address
		c.mu.Lock()
		c.endpoints[msg.FromPeer] = &PeerEndpoint{
			PeerID:     msg.FromPeer,
			PublicIP:   from.IP.String(),
			PublicPort: from.Port,
			LastSeen:   time.Now(),
		}
		c.mu.Unlock()

		// Send back their public address
		resp := map[string]interface{}{
			"type":      MsgPong,
			"your_ip":   from.IP.String(),
			"your_port": from.Port,
			"timestamp": time.Now().UnixNano(),
		}
		data, _ := json.Marshal(resp)
		c.conn.WriteToUDP(data, from)

		log.Printf("[HolePunch Coordinator] Peer %s registered: %s:%d", msg.FromPeer, from.IP, from.Port)

	case "get_endpoint":
		// Peer requesting another peer's endpoint for hole punching
		c.mu.RLock()
		targetEP, exists := c.endpoints[msg.ToPeer]
		c.mu.RUnlock()

		if !exists {
			resp := map[string]interface{}{
				"type":  "error",
				"error": "peer not found",
			}
			data, _ := json.Marshal(resp)
			c.conn.WriteToUDP(data, from)
			return
		}

		resp := map[string]interface{}{
			"type":      "endpoint",
			"peer_id":   targetEP.PeerID,
			"ip":        targetEP.PublicIP,
			"port":      targetEP.PublicPort,
			"timestamp": time.Now().UnixNano(),
		}
		data, _ := json.Marshal(resp)
		c.conn.WriteToUDP(data, from)
	}
}

// cleanupLoop removes stale endpoints
func (c *Coordinator) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for id, ep := range c.endpoints {
				if now.Sub(ep.LastSeen) > 2*time.Minute {
					delete(c.endpoints, id)
					log.Printf("[HolePunch Coordinator] Removed stale endpoint: %s", id)
				}
			}
			c.mu.Unlock()
		}
	}
}

