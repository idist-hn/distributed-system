# DHT (Distributed Hash Table) - Kademlia

## Tổng quan

**Kademlia DHT** cho phép peers tìm thấy nhau và lưu trữ dữ liệu phân tán mà không cần tracker trung tâm.

## Kiến trúc

```
┌─────────────────────────────────────────────────────────────────┐
│                    KADEMLIA DHT NETWORK                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│     Node A                Node B                Node C           │
│   ┌───────┐            ┌───────┐            ┌───────┐           │
│   │ID: 001│◄──────────►│ID: 010│◄──────────►│ID: 100│           │
│   │       │            │       │            │       │           │
│   │Bucket │            │Bucket │            │Bucket │           │
│   │[010]  │            │[001]  │            │[010]  │           │
│   │[100]  │            │[100]  │            │[001]  │           │
│   └───────┘            └───────┘            └───────┘           │
│       │                    │                    │                │
│       └────────────────────┼────────────────────┘                │
│                            │                                     │
│                    ┌───────▼───────┐                            │
│                    │  XOR Distance  │                            │
│                    │   Metric       │                            │
│                    └───────────────┘                            │
└─────────────────────────────────────────────────────────────────┘
```

## Các khái niệm chính

### 1. Node ID
- 256-bit identifier (SHA-256 hash)
- Unique cho mỗi node trong mạng

### 2. XOR Distance
- Khoảng cách giữa 2 nodes = XOR của IDs
- Symmetric: d(A,B) = d(B,A)
- Triangle inequality: d(A,C) ≤ d(A,B) + d(B,C)

### 3. K-Buckets
- Mỗi node có 256 buckets (1 cho mỗi bit)
- Bucket i chứa nodes có distance 2^i đến 2^(i+1)
- Mỗi bucket chứa tối đa K=20 nodes

### 4. Routing Table
```
┌─────────────────────────────────────────┐
│           ROUTING TABLE                  │
├─────────────────────────────────────────┤
│ Bucket 0: [nodes with prefix 0...]      │
│ Bucket 1: [nodes with prefix 10...]     │
│ Bucket 2: [nodes with prefix 110...]    │
│ ...                                      │
│ Bucket 255: [nodes with prefix 1111...] │
└─────────────────────────────────────────┘
```

## RPC Operations

| Operation | Description |
|-----------|-------------|
| `PING` | Kiểm tra node còn sống |
| `FIND_NODE` | Tìm K nodes gần nhất với target ID |
| `STORE` | Lưu key-value pair |
| `FIND_VALUE` | Tìm value theo key |

## API

### Node Creation

```go
// Create DHT node
transport := NewUDPTransport(":6881")
node := dht.NewNode("192.168.1.1:6881", transport)

// Bootstrap with known nodes
bootstrapNodes := []*dht.NodeInfo{
    {ID: dht.NewNodeID("node1"), Address: "1.2.3.4:6881"},
    {ID: dht.NewNodeID("node2"), Address: "5.6.7.8:6881"},
}
node.Bootstrap(bootstrapNodes)
```

### Store and Retrieve

```go
// Store file info in DHT
fileHash := "abc123..."
peerInfo := []byte(`{"peer_id":"xyz","address":"1.2.3.4:6881"}`)
node.Store(fileHash, peerInfo)

// Find peers with file
data, err := node.Get(fileHash)
```

### Node Lookup

```go
// Find closest nodes to a target
targetID := dht.NewNodeID("target-hash")
closestNodes := node.GetClosestNodes(targetID)
```

## Lookup Algorithm

```
┌─────────────────────────────────────────────────────────────────┐
│                    ITERATIVE LOOKUP                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. Get α closest nodes from local routing table                │
│                                                                  │
│  2. Send FIND_NODE to α nodes in parallel                       │
│                                                                  │
│  3. Receive responses with closer nodes                         │
│                                                                  │
│  4. Merge results, keep K closest                               │
│                                                                  │
│  5. Repeat until no closer nodes found                          │
│                                                                  │
│  6. Return K closest nodes                                      │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Parameters

| Parameter | Value | Description |
|-----------|-------|-------------|
| K | 20 | Replication factor (bucket size) |
| α (Alpha) | 3 | Concurrency parameter |
| ID Length | 256 bits | Node ID size |

## Use Cases trong P2P

### 1. Peer Discovery
```go
// Announce file availability
fileHash := hashFile(file)
myInfo := json.Marshal(PeerInfo{ID: myID, Address: myAddr})
node.Store(fileHash, myInfo)

// Find peers with file
data, _ := node.Get(fileHash)
var peers []PeerInfo
json.Unmarshal(data, &peers)
```

### 2. Decentralized Tracker
```go
// No central tracker needed
// Each peer stores and retrieves peer lists via DHT
```

### 3. Content Addressing
```go
// Store content by hash
contentHash := sha256(content)
node.Store(hex.EncodeToString(contentHash), content)
```

## Ví dụ Complete

```go
// Node A joins network
nodeA := dht.NewNode("192.168.1.1:6881", transport)
nodeA.Bootstrap(bootstrapNodes)

// Node A announces file
fileHash := "abc123"
nodeA.Store(fileHash, []byte(`{"peer":"A","addr":"192.168.1.1:6881"}`))

// Node B looks up file
nodeB := dht.NewNode("192.168.1.2:6881", transport)
nodeB.Bootstrap(bootstrapNodes)

data, _ := nodeB.Get(fileHash)
// data = {"peer":"A","addr":"192.168.1.1:6881"}
```

## Lưu ý

1. **Bootstrap nodes**: Cần ít nhất 1 node đã biết để join network
2. **Republishing**: Values cần được republish định kỳ (mỗi 1 giờ)
3. **Bucket refresh**: Buckets cần được refresh nếu không có activity
4. **NAT traversal**: DHT không giải quyết NAT, cần kết hợp với hole punching

