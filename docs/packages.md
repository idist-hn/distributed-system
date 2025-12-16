# 📦 Package Documentation

Tài liệu mô tả chi tiết các packages trong hệ thống P2P File Sharing.

## Overview

```
pkg/
├── chunker/        # File chunking (256KB)
├── crypto/         # E2E encryption
├── dht/            # Kademlia DHT
├── hash/           # SHA-256 hashing
├── holepunch/      # UDP NAT hole punching
├── logger/         # Structured logging
├── magnet/         # Magnet URI parsing & generation
├── merkle/         # Merkle tree verification
├── peerscore/      # Peer scoring & selection
├── pieceselection/ # Smart piece selection algorithms
├── protocol/       # Message definitions
└── throttle/       # Bandwidth limiting
```

---

## 📁 pkg/chunker

**Chức năng**: Chia file thành các chunks có kích thước cố định.

### API

```go
// ChunkFile chia file thành chunks
func ChunkFile(filePath string, chunkSize int) ([]Chunk, error)

// Chunk represents một phần của file
type Chunk struct {
    Index int
    Data  []byte
    Hash  string
    Size  int
}

// Default chunk size: 256KB
const DefaultChunkSize = 256 * 1024
```

### Usage

```go
chunks, err := chunker.ChunkFile("myfile.zip", chunker.DefaultChunkSize)
for _, chunk := range chunks {
    fmt.Printf("Chunk %d: %s (%d bytes)\n", chunk.Index, chunk.Hash, chunk.Size)
}
```

---

## 🔐 pkg/crypto

**Chức năng**: End-to-end encryption cho P2P communication.

### Components

| File | Chức năng |
|------|-----------|
| `crypto.go` | X25519 key generation, ECDH |
| `session.go` | Encrypted session management |

### API

```go
// Generate key pair
pub, priv, _ := crypto.GenerateKeyPair()

// Derive shared secret
sharedSecret := crypto.DeriveSharedSecret(myPrivKey, theirPubKey)

// Create encrypted session
session := crypto.NewSession(sharedSecret)

// Encrypt/Decrypt
ciphertext := session.Encrypt(plaintext)
plaintext := session.Decrypt(ciphertext)
```

### Algorithms
- **Key Exchange**: X25519 ECDH
- **Encryption**: AES-256-GCM
- **Key Derivation**: HKDF-SHA256

---

## 🌐 pkg/dht

**Chức năng**: Kademlia Distributed Hash Table cho peer discovery.

### Components

| File | Chức năng |
|------|-----------|
| `node.go` | DHT node implementation |
| `routing.go` | Routing table (k-buckets) |

### API

```go
// Create DHT node
node := dht.NewNode(nodeID, listenAddr)
node.Start()

// Bootstrap
node.Bootstrap(bootstrapAddr)

// Store and retrieve
node.Store(key, value)
value, _ := node.FindValue(key)

// Find peers
peers := node.FindNode(targetID)
```

### Parameters
- **K (Bucket Size)**: 20
- **Alpha (Concurrency)**: 3
- **ID Length**: 160 bits (SHA-1)

---

## #️⃣ pkg/hash

**Chức năng**: SHA-256 hashing utilities.

### API

```go
// Hash bytes
hash := hash.SHA256(data)

// Hash file
fileHash, _ := hash.HashFile(filePath)

// Verify hash
isValid := hash.Verify(data, expectedHash)
```

---

## 🔓 pkg/holepunch

**Chức năng**: UDP NAT hole punching cho direct P2P connections.

### Components

| File | Chức năng |
|------|-----------|
| `holepunch.go` | Peer-side puncher |
| `coordinator.go` | Tracker-side coordinator |

### API

```go
// Peer side
puncher, _ := holepunch.NewPuncher(peerID, 0)
puncher.Start()

// Discover public address
puncher.DiscoverPublicAddress(trackerAddr)

// Punch to peer
puncher.PunchTo(ctx, targetPeerID, targetEndpoint)

// Send data
puncher.SendTo(peerID, data)

// Tracker side
coordinator, _ := holepunch.NewCoordinator(9999)
coordinator.Start()
```

### Message Types
| Type | Description |
|------|-------------|
| `punch` | Initiate hole punch |
| `punch_ack` | Acknowledge success |
| `data` | Application data |
| `ping` | Registration/keepalive |
| `pong` | Ping response |

---

## 📝 pkg/logger

**Chức năng**: Structured logging with levels.

### API

```go
logger.Info("Server started", "port", 8080)
logger.Error("Connection failed", "error", err)
logger.Debug("Processing chunk", "index", 5)
```

---

## 🌳 pkg/merkle

**Chức năng**: Merkle tree cho efficient chunk verification.

### API

```go
// Build tree from chunk hashes
tree := merkle.NewTree(chunkHashes)

// Get root hash
root := tree.Root()

// Generate proof for chunk
proof := tree.GenerateProof(chunkIndex)

// Verify chunk with proof
isValid := merkle.VerifyProof(chunkHash, proof, root)
```

### Structure

```
              Root Hash
             /         \
        Hash01          Hash23
       /     \         /     \
   Hash0   Hash1   Hash2   Hash3
     │       │       │       │
  Chunk0  Chunk1  Chunk2  Chunk3
```

---

## 📨 pkg/protocol

**Chức năng**: Message definitions cho P2P communication.

### Message Types

```go
const (
    MsgHandshake  = "handshake"
    MsgBitfield   = "bitfield"
    MsgRequest    = "request"
    MsgPiece      = "piece"
    MsgHave       = "have"
    MsgChoke      = "choke"
    MsgUnchoke    = "unchoke"
    MsgInterested = "interested"
)
```

### Message Structure

```go
type Message struct {
    Type    string      `json:"type"`
    Payload interface{} `json:"payload"`
}

type RequestPayload struct {
    FileHash   string `json:"file_hash"`
    ChunkIndex int    `json:"chunk_index"`
}

type PiecePayload struct {
    FileHash   string `json:"file_hash"`
    ChunkIndex int    `json:"chunk_index"`
    Data       []byte `json:"data"`
    Hash       string `json:"hash"`
}
```

---

## ⏱️ pkg/throttle

**Chức năng**: Bandwidth limiting với token bucket algorithm.

### API

```go
// Create limiter (1 MB/s)
limiter := throttle.NewLimiter(1024 * 1024)

// Wait for tokens before sending
limiter.Wait(len(data))

// Or use reader/writer wrappers
reader := throttle.NewReader(file, limiter)
writer := throttle.NewWriter(conn, limiter)
```

### Configuration

```go
type Limiter struct {
    BytesPerSecond int64  // Rate limit
    BurstSize      int64  // Max burst (default: 1 second worth)
}
```

---

## 🧪 Test Coverage

| Package | Tests | Coverage |
|---------|-------|----------|
| `chunker` | ✅ | ~80% |
| `hash` | ✅ | ~90% |
| `merkle` | ✅ 11 tests | ~85% |
| `throttle` | ✅ 11 tests | ~85% |
| `holepunch` | ✅ 13 tests | ~75% |
| `crypto` | 📋 Planned | - |
| `dht` | 📋 Planned | - |

### Run Tests

```bash
# All packages
go test -v ./pkg/...

# Specific package
go test -v ./pkg/merkle/...

# With coverage
go test -cover ./pkg/...
```

---

## 🎯 pkg/pieceselection

**Chức năng**: Smart piece selection algorithms cho P2P downloads.

### Algorithms

| Algorithm | Description | Use Case |
|-----------|-------------|----------|
| Rarest First | Download rarest pieces first | Improve swarm health |
| Random First | Random piece selection | Initial bootstrap |
| Sequential | Download in order | Streaming |

### API

```go
// Selector interface
type Selector interface {
    SelectNext(pieces []PieceInfo, availablePeers []string) (pieceIndex int, peerID string, ok bool)
    Name() string
}

// Create selectors
selector := pieceselection.NewRarestFirstSelector()
selector := pieceselection.NewRandomFirstSelector()
selector := pieceselection.NewSequentialSelector()

// Select next piece
pieceIdx, peerID, ok := selector.SelectNext(pieces, peers)
```

---

## 📊 pkg/peerscore

**Chức năng**: Peer scoring và selection dựa trên performance metrics.

### Scoring Components

| Component | Weight | Description |
|-----------|--------|-------------|
| Download Speed | 30% | Bytes/second |
| Upload Ratio | 20% | Tit-for-tat fairness |
| Reliability | 25% | Success rate |
| Latency | 15% | Response time |
| Recency | 10% | Recent activity |

### API

```go
// Create scorer
scorer := peerscore.NewScorer(peerscore.DefaultConfig())

// Record activity
scorer.RecordDownload("peer1", 1024*1024, 50*time.Millisecond)
scorer.RecordUpload("peer1", 512*1024)
scorer.RecordFailure("peer1")

// Get score
score := scorer.GetScore("peer1")
fmt.Printf("Score: %.2f\n", score.TotalScore)

// Get top peers
topPeers := scorer.GetTopPeers(10)
```

---

## 🧲 pkg/magnet

**Chức năng**: Magnet URI parsing và generation cho file sharing.

### Magnet URI Format

```
magnet:?xt=urn:sha256:<hash>&dn=<name>&xl=<size>&tr=<tracker>&x.cs=<chunksize>&x.tc=<totalchunks>
```

### API

```go
// Parse magnet URI
m, err := magnet.Parse("magnet:?xt=urn:sha256:abc123&dn=file.txt&xl=1024")

// Create new magnet
m := magnet.New("abc123def456", "myfile.zip", 10485760)
m.AddTracker("https://tracker.example.com")
m.SetChunkInfo(262144, 40)

// Generate URI
uri := m.String()
// Output: magnet:?xt=urn:sha256:abc123def456&dn=myfile.zip&xl=10485760&tr=...
```

### Fields

| Field | Parameter | Description |
|-------|-----------|-------------|
| InfoHash | `xt` | File hash (SHA-256) |
| DisplayName | `dn` | File name |
| Size | `xl` | File size in bytes |
| Trackers | `tr` | Tracker URLs (multiple) |
| ChunkSize | `x.cs` | Chunk size (custom extension) |
| TotalChunks | `x.tc` | Total chunks (custom extension) |
