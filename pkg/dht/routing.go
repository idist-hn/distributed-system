package dht

import (
	"sort"
	"sync"
	"time"
)

// Bucket represents a k-bucket in the routing table
type Bucket struct {
	nodes []*NodeInfo
	mu    sync.RWMutex
}

// NewBucket creates a new bucket
func NewBucket() *Bucket {
	return &Bucket{
		nodes: make([]*NodeInfo, 0, K),
	}
}

// AddNode adds a node to the bucket
func (b *Bucket) AddNode(node *NodeInfo) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check if node already exists
	for i, n := range b.nodes {
		if n.ID == node.ID {
			// Move to end (most recently seen)
			b.nodes = append(b.nodes[:i], b.nodes[i+1:]...)
			node.LastSeen = time.Now()
			b.nodes = append(b.nodes, node)
			return true
		}
	}

	// Add new node if bucket not full
	if len(b.nodes) < K {
		node.LastSeen = time.Now()
		b.nodes = append(b.nodes, node)
		return true
	}

	// Bucket is full, check if oldest node is stale
	// For now, just reject (in production, ping oldest node first)
	return false
}

// RemoveNode removes a node from the bucket
func (b *Bucket) RemoveNode(id NodeID) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, n := range b.nodes {
		if n.ID == id {
			b.nodes = append(b.nodes[:i], b.nodes[i+1:]...)
			return
		}
	}
}

// GetNodes returns all nodes in the bucket
func (b *Bucket) GetNodes() []*NodeInfo {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]*NodeInfo, len(b.nodes))
	copy(result, b.nodes)
	return result
}

// Len returns the number of nodes in the bucket
func (b *Bucket) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.nodes)
}

// RoutingTable manages k-buckets for node routing
type RoutingTable struct {
	localID NodeID
	buckets [IDLength]*Bucket
	mu      sync.RWMutex
}

// NewRoutingTable creates a new routing table
func NewRoutingTable(localID NodeID) *RoutingTable {
	rt := &RoutingTable{
		localID: localID,
	}
	for i := 0; i < IDLength; i++ {
		rt.buckets[i] = NewBucket()
	}
	return rt
}

// getBucketIndex returns the bucket index for a node ID
func (rt *RoutingTable) getBucketIndex(id NodeID) int {
	dist := rt.localID.Distance(id)
	prefixLen := dist.PrefixLen()
	if prefixLen >= IDLength {
		return IDLength - 1
	}
	return prefixLen
}

// AddNode adds a node to the appropriate bucket
func (rt *RoutingTable) AddNode(node *NodeInfo) bool {
	if node.ID == rt.localID {
		return false // Don't add ourselves
	}

	index := rt.getBucketIndex(node.ID)
	return rt.buckets[index].AddNode(node)
}

// RemoveNode removes a node from the routing table
func (rt *RoutingTable) RemoveNode(id NodeID) {
	index := rt.getBucketIndex(id)
	rt.buckets[index].RemoveNode(id)
}

// FindClosest returns the n closest nodes to a target ID
func (rt *RoutingTable) FindClosest(target NodeID, n int) []*NodeInfo {
	var allNodes []*NodeInfo

	// Collect all nodes from all buckets
	for _, bucket := range rt.buckets {
		allNodes = append(allNodes, bucket.GetNodes()...)
	}

	// Sort by distance to target
	sort.Slice(allNodes, func(i, j int) bool {
		distI := allNodes[i].ID.Distance(target)
		distJ := allNodes[j].ID.Distance(target)
		return distI.Less(distJ)
	})

	// Return top n
	if len(allNodes) > n {
		return allNodes[:n]
	}
	return allNodes
}

// GetAllNodes returns all nodes in the routing table
func (rt *RoutingTable) GetAllNodes() []*NodeInfo {
	var allNodes []*NodeInfo
	for _, bucket := range rt.buckets {
		allNodes = append(allNodes, bucket.GetNodes()...)
	}
	return allNodes
}

// Size returns the total number of nodes in the routing table
func (rt *RoutingTable) Size() int {
	count := 0
	for _, bucket := range rt.buckets {
		count += bucket.Len()
	}
	return count
}

