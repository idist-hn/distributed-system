// Package dht implements a Kademlia-based Distributed Hash Table
package dht

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"
)

const (
	// K is the replication parameter (bucket size)
	K = 20
	// Alpha is the concurrency parameter
	Alpha = 3
	// IDLength is the length of node IDs in bits
	IDLength = 256
	// IDBytes is the length of node IDs in bytes
	IDBytes = IDLength / 8
)

// NodeID represents a 256-bit node identifier
type NodeID [IDBytes]byte

// NodeInfo contains information about a node in the DHT
type NodeInfo struct {
	ID       NodeID    `json:"id"`
	Address  string    `json:"address"` // IP:Port
	LastSeen time.Time `json:"last_seen"`
}

// Node represents a DHT node
type Node struct {
	ID           NodeID
	Address      string
	routingTable *RoutingTable
	storage      map[string][]byte // key -> value storage
	storageMu    sync.RWMutex
	transport    Transport
	mu           sync.RWMutex
}

// Transport interface for network communication
type Transport interface {
	SendPing(target *NodeInfo) error
	SendFindNode(target *NodeInfo, nodeID NodeID) ([]*NodeInfo, error)
	SendStore(target *NodeInfo, key string, value []byte) error
	SendFindValue(target *NodeInfo, key string) ([]byte, []*NodeInfo, error)
}

// NewNodeID creates a NodeID from a string (hashed)
func NewNodeID(s string) NodeID {
	hash := sha256.Sum256([]byte(s))
	var id NodeID
	copy(id[:], hash[:])
	return id
}

// NewNodeIDFromBytes creates a NodeID from bytes
func NewNodeIDFromBytes(b []byte) NodeID {
	var id NodeID
	copy(id[:], b)
	return id
}

// String returns hex representation of NodeID
func (id NodeID) String() string {
	return hex.EncodeToString(id[:])
}

// Distance calculates XOR distance between two NodeIDs
func (id NodeID) Distance(other NodeID) NodeID {
	var dist NodeID
	for i := 0; i < IDBytes; i++ {
		dist[i] = id[i] ^ other[i]
	}
	return dist
}

// PrefixLen returns the number of leading zero bits (common prefix length)
func (id NodeID) PrefixLen() int {
	for i := 0; i < IDBytes; i++ {
		for j := 7; j >= 0; j-- {
			if (id[i]>>j)&1 != 0 {
				return i*8 + (7 - j)
			}
		}
	}
	return IDLength
}

// Less compares two NodeIDs
func (id NodeID) Less(other NodeID) bool {
	for i := 0; i < IDBytes; i++ {
		if id[i] < other[i] {
			return true
		}
		if id[i] > other[i] {
			return false
		}
	}
	return false
}

// NewNode creates a new DHT node
func NewNode(address string, transport Transport) *Node {
	id := NewNodeID(address)
	return &Node{
		ID:           id,
		Address:      address,
		routingTable: NewRoutingTable(id),
		storage:      make(map[string][]byte),
		transport:    transport,
	}
}

// Bootstrap joins the DHT network using bootstrap nodes
func (n *Node) Bootstrap(bootstrapNodes []*NodeInfo) error {
	// Add bootstrap nodes to routing table
	for _, node := range bootstrapNodes {
		n.routingTable.AddNode(node)
	}

	// Perform node lookup on our own ID to populate routing table
	_, err := n.FindNode(n.ID)
	return err
}

// FindNode finds the K closest nodes to a target ID
func (n *Node) FindNode(targetID NodeID) ([]*NodeInfo, error) {
	// Get initial closest nodes from routing table
	closest := n.routingTable.FindClosest(targetID, Alpha)
	if len(closest) == 0 {
		return nil, fmt.Errorf("no nodes in routing table")
	}

	queried := make(map[string]bool)
	var result []*NodeInfo

	for {
		// Query Alpha closest unqueried nodes
		var toQuery []*NodeInfo
		for _, node := range closest {
			if !queried[node.ID.String()] {
				toQuery = append(toQuery, node)
				if len(toQuery) >= Alpha {
					break
				}
			}
		}

		if len(toQuery) == 0 {
			break
		}

		// Query nodes in parallel
		var wg sync.WaitGroup
		var mu sync.Mutex
		var newNodes []*NodeInfo

		for _, node := range toQuery {
			queried[node.ID.String()] = true
			wg.Add(1)
			go func(target *NodeInfo) {
				defer wg.Done()
				nodes, err := n.transport.SendFindNode(target, targetID)
				if err == nil {
					mu.Lock()
					newNodes = append(newNodes, nodes...)
					n.routingTable.AddNode(target)
					mu.Unlock()
				}
			}(node)
		}
		wg.Wait()

		// Merge and sort results
		closest = n.mergeAndSort(closest, newNodes, targetID)
		if len(closest) > K {
			closest = closest[:K]
		}
	}

	result = closest
	return result, nil
}

// Store stores a key-value pair in the DHT
func (n *Node) Store(key string, value []byte) error {
	keyID := NewNodeID(key)

	// Find K closest nodes to the key
	closest, err := n.FindNode(keyID)
	if err != nil {
		return err
	}

	// Store on all K closest nodes
	var wg sync.WaitGroup
	var storeErr error
	var errMu sync.Mutex

	for _, node := range closest {
		wg.Add(1)
		go func(target *NodeInfo) {
			defer wg.Done()
			if err := n.transport.SendStore(target, key, value); err != nil {
				errMu.Lock()
				storeErr = err
				errMu.Unlock()
			}
		}(node)
	}
	wg.Wait()

	// Also store locally
	n.storageMu.Lock()
	n.storage[key] = value
	n.storageMu.Unlock()

	return storeErr
}

// Get retrieves a value from the DHT
func (n *Node) Get(key string) ([]byte, error) {
	// Check local storage first
	n.storageMu.RLock()
	if value, exists := n.storage[key]; exists {
		n.storageMu.RUnlock()
		return value, nil
	}
	n.storageMu.RUnlock()

	keyID := NewNodeID(key)

	// Get initial closest nodes
	closest := n.routingTable.FindClosest(keyID, Alpha)
	if len(closest) == 0 {
		return nil, fmt.Errorf("key not found")
	}

	queried := make(map[string]bool)

	for {
		var toQuery []*NodeInfo
		for _, node := range closest {
			if !queried[node.ID.String()] {
				toQuery = append(toQuery, node)
				if len(toQuery) >= Alpha {
					break
				}
			}
		}

		if len(toQuery) == 0 {
			break
		}

		// Query nodes
		for _, node := range toQuery {
			queried[node.ID.String()] = true
			value, nodes, err := n.transport.SendFindValue(node, key)
			if err == nil {
				if value != nil {
					return value, nil
				}
				closest = n.mergeAndSort(closest, nodes, keyID)
				if len(closest) > K {
					closest = closest[:K]
				}
			}
		}
	}

	return nil, fmt.Errorf("key not found")
}

// LocalStore stores a value locally
func (n *Node) LocalStore(key string, value []byte) {
	n.storageMu.Lock()
	defer n.storageMu.Unlock()
	n.storage[key] = value
}

// LocalGet retrieves a value from local storage
func (n *Node) LocalGet(key string) ([]byte, bool) {
	n.storageMu.RLock()
	defer n.storageMu.RUnlock()
	value, exists := n.storage[key]
	return value, exists
}

// AddNode adds a node to the routing table
func (n *Node) AddNode(node *NodeInfo) {
	n.routingTable.AddNode(node)
}

// GetClosestNodes returns K closest nodes to a target
func (n *Node) GetClosestNodes(targetID NodeID) []*NodeInfo {
	return n.routingTable.FindClosest(targetID, K)
}

// mergeAndSort merges two node lists and sorts by distance
func (n *Node) mergeAndSort(a, b []*NodeInfo, target NodeID) []*NodeInfo {
	seen := make(map[string]bool)
	var result []*NodeInfo

	for _, node := range a {
		if !seen[node.ID.String()] {
			seen[node.ID.String()] = true
			result = append(result, node)
		}
	}
	for _, node := range b {
		if !seen[node.ID.String()] {
			seen[node.ID.String()] = true
			result = append(result, node)
		}
	}

	// Sort by distance to target
	sort.Slice(result, func(i, j int) bool {
		distI := result[i].ID.Distance(target)
		distJ := result[j].ID.Distance(target)
		return distI.Less(distJ)
	})

	return result
}
