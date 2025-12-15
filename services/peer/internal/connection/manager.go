package connection

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/p2p-filesharing/distributed-system/services/peer/internal/p2p"
	"github.com/p2p-filesharing/distributed-system/services/peer/internal/relay"
)

// ConnectionType represents the type of connection
type ConnectionType int

const (
	ConnTypeDirect ConnectionType = iota
	ConnTypeRelay
)

// PeerInfo contains information about a remote peer
type PeerInfo struct {
	ID   string
	IP   string
	Port int
}

// Manager manages connections to remote peers
type Manager struct {
	peerID      string
	p2pClient   *p2p.Client
	relayClient *relay.Client
	connections map[string]*PeerConnection
	mu          sync.RWMutex
}

// PeerConnection represents a connection to a peer
type PeerConnection struct {
	PeerInfo   PeerInfo
	ConnType   ConnectionType
	DirectConn *p2p.PeerConnection
	LastUsed   time.Time
}

// NewManager creates a new connection manager
func NewManager(peerID string, p2pClient *p2p.Client, relayClient *relay.Client) *Manager {
	return &Manager{
		peerID:      peerID,
		p2pClient:   p2pClient,
		relayClient: relayClient,
		connections: make(map[string]*PeerConnection),
	}
}

// RequestChunk requests a chunk from a peer, trying direct then relay
func (m *Manager) RequestChunk(peer PeerInfo, fileHash string, chunkIndex int, expectedHash string) ([]byte, error) {
	// Try to get existing connection
	m.mu.RLock()
	conn, exists := m.connections[peer.ID]
	m.mu.RUnlock()

	if exists && conn.DirectConn != nil {
		// Try existing direct connection
		data, err := conn.DirectConn.RequestChunk(fileHash, chunkIndex, expectedHash)
		if err == nil {
			m.updateLastUsed(peer.ID)
			return data, nil
		}
		// Connection failed, remove it
		log.Printf("[ConnMgr] Direct connection to %s failed: %v", peer.ID, err)
		m.removeConnection(peer.ID)
	}

	// Strategy 1: Try direct TCP connection
	data, err := m.tryDirectConnection(peer, fileHash, chunkIndex, expectedHash)
	if err == nil {
		return data, nil
	}
	log.Printf("[ConnMgr] Direct connection failed: %v, trying relay...", err)

	// Strategy 2: Try relay connection
	if m.relayClient != nil && m.relayClient.IsConnected() {
		data, err = m.tryRelayConnection(peer, fileHash, chunkIndex)
		if err == nil {
			return data, nil
		}
		log.Printf("[ConnMgr] Relay connection failed: %v", err)
	}

	return nil, fmt.Errorf("all connection methods failed for peer %s", peer.ID)
}

// tryDirectConnection attempts a direct TCP connection
func (m *Manager) tryDirectConnection(peer PeerInfo, fileHash string, chunkIndex int, expectedHash string) ([]byte, error) {
	// Try to establish direct connection with timeout
	conn, err := m.p2pClient.Connect(peer.IP, peer.Port)
	if err != nil {
		return nil, fmt.Errorf("direct connect failed: %w", err)
	}

	// Save connection for reuse
	m.mu.Lock()
	m.connections[peer.ID] = &PeerConnection{
		PeerInfo:   peer,
		ConnType:   ConnTypeDirect,
		DirectConn: conn,
		LastUsed:   time.Now(),
	}
	m.mu.Unlock()

	// Request chunk
	data, err := conn.RequestChunk(fileHash, chunkIndex, expectedHash)
	if err != nil {
		m.removeConnection(peer.ID)
		return nil, err
	}

	return data, nil
}

// tryRelayConnection attempts connection via relay server
func (m *Manager) tryRelayConnection(peer PeerInfo, fileHash string, chunkIndex int) ([]byte, error) {
	if m.relayClient == nil {
		return nil, fmt.Errorf("relay client not available")
	}

	data, err := m.relayClient.RequestChunk(peer.ID, fileHash, chunkIndex)
	if err != nil {
		return nil, fmt.Errorf("relay request failed: %w", err)
	}

	// Store relay connection info
	m.mu.Lock()
	m.connections[peer.ID] = &PeerConnection{
		PeerInfo: peer,
		ConnType: ConnTypeRelay,
		LastUsed: time.Now(),
	}
	m.mu.Unlock()

	return data, nil
}

// updateLastUsed updates the last used time for a connection
func (m *Manager) updateLastUsed(peerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if conn, ok := m.connections[peerID]; ok {
		conn.LastUsed = time.Now()
	}
}

// removeConnection removes a connection from the manager
func (m *Manager) removeConnection(peerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if conn, ok := m.connections[peerID]; ok {
		if conn.DirectConn != nil {
			conn.DirectConn.Close()
		}
		delete(m.connections, peerID)
	}
}

// Close closes all connections
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, conn := range m.connections {
		if conn.DirectConn != nil {
			conn.DirectConn.Close()
		}
	}
	m.connections = make(map[string]*PeerConnection)
}

