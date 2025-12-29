# Kiến Trúc Hệ Thống P2P File Sharing

## 1. Tổng Quan

Hệ thống chia sẻ file ngang hàng (P2P) cho phép các peer trao đổi file trực tiếp với nhau. Sử dụng mô hình **Hybrid P2P** với Tracker đóng vai trò điều phối và relay cho NAT traversal.

**Tính năng chính:**
- Smart connection strategy: Direct TCP (5s) → WebSocket Relay
- PostgreSQL persistent storage
- Parallel chunk downloads với peer scoring
- Real-time monitoring qua WebSocket

## 2. Kiến Trúc Tổng Thể

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           TRACKER SERVER                                 │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                         HTTP Server (:8080)                         │ │
│  ├──────────────┬──────────────┬──────────────┬───────────────────────┤ │
│  │  REST API    │  WebSocket   │  Dashboard   │  Relay Hub            │ │
│  │  /api/*      │  /ws         │  /dashboard  │  /relay               │ │
│  └──────────────┴──────────────┴──────────────┴───────────────────────┘ │
│                                    │                                     │
│  ┌────────────────────────────────▼────────────────────────────────────┐│
│  │                       PostgreSQL Storage                             ││
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                  ││
│  │  │ peers       │  │ files       │  │ chunks      │                  ││
│  │  │ - id, addr  │  │ - hash      │  │ - file_hash │                  ││
│  │  │ - status    │  │ - name,size │  │ - index     │                  ││
│  │  │ - last_seen │  │ - seeders   │  │ - hash      │                  ││
│  │  └─────────────┘  └─────────────┘  └─────────────┘                  ││
│  └─────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────┘
```
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│    PEER A     │     │    PEER B     │     │    PEER C     │
│  (Seeder)     │     │  (Leecher)    │     │  (Seeder)     │
├───────────────┤     ├───────────────┤     ├───────────────┤
│┌─────────────┐│     │┌─────────────┐│     │┌─────────────┐│
││ TCP Server  ││     ││ TCP Server  ││     ││ TCP Server  ││
││   :6881     ││     ││   :6882     ││     ││   :6883     ││
│└─────────────┘│     │└─────────────┘│     │└─────────────┘│
│┌─────────────┐│     │┌─────────────┐│     │┌─────────────┐│
││ Connection  ││     ││ Connection  ││     ││ Connection  ││
││  Manager    ││     ││  Manager    ││     ││  Manager    ││
│└─────────────┘│     │└─────────────┘│     │└─────────────┘│
│┌─────────────┐│     │┌─────────────┐│     │┌─────────────┐│
││  Storage    ││     ││  Storage    ││     ││  Storage    ││
││  ./data/    ││     ││  ./data/    ││     ││  ./data/    ││
│└─────────────┘│     │└─────────────┘│     │└─────────────┘│
└───────────────┘     └───────────────┘     └───────────────┘
        ▲                     │                     ▲
        └──────────── Direct P2P Transfer ─────────┘
```

## 3. Thành Phần Chi Tiết

### 3.1 Tracker Server

| Module | File | Chức năng |
|--------|------|-----------|
| REST API | `api/handlers.go` | CRUD peers, files |
| WebSocket | `api/websocket.go` | Real-time events |
| Relay Hub | `api/relay.go` | NAT traversal relay |
| Dashboard | `api/dashboard.go` | Web UI monitoring |
| Middleware | `api/middleware.go` | Auth, logging, metrics |
| Storage | `storage/storage.go` | In-memory data store |

### 3.2 Peer Node

| Module | File | Chức năng |
|--------|------|-----------|
| Tracker Client | `client/tracker.go` | API communication |
| P2P Server | `p2p/server.go` | TCP listener for chunks |
| P2P Client | `p2p/client.go` | Request chunks from peers |
| Connection Manager | `connection/manager.go` | Strategy: Direct→Punch→Relay |
| Downloader | `downloader/downloader.go` | Parallel chunk downloads |
| Relay Client | `relay/client.go` | WebSocket relay tunnel |
| Storage | `storage/local.go` | File & chunk management |

### 3.3 Shared Packages (pkg/)

| Package | Chức năng |
|---------|-----------|
| `chunker` | Chia file thành chunks 256KB |
| `hash` | SHA-256 hashing |
| `protocol` | Message definitions |
| `crypto` | E2E encryption (X25519 + AES-GCM) |
| `dht` | Kademlia DHT |
| `holepunch` | UDP NAT hole punching |
| `merkle` | Merkle tree verification |
| `throttle` | Bandwidth limiting |
| `logger` | Structured logging |

## 4. Connection Strategy

```
┌──────────────────────────────────────────────────────────────┐
│                    Smart Connection Strategy                  │
├──────────────────────────────────────────────────────────────┤
│  Step 1: Try Direct TCP (5 second timeout)                   │
│  ┌──────────┐         ┌──────────┐                           │
│  │  Peer A  │ ──TCP──▶│  Peer B  │   ✓ Fastest               │
│  └──────────┘         └──────────┘   ✓ No overhead           │
│                                                               │
│  If success → Use direct TCP for all chunks                  │
│  If timeout → Switch to relay-only mode                      │
├──────────────────────────────────────────────────────────────┤
│  Step 2: WebSocket Relay (if Direct fails)                   │
│  ┌──────────┐   WS    ┌──────────┐    WS   ┌──────────┐      │
│  │  Peer A  │────────▶│ Tracker  │◀────────│  Peer B  │      │
│  └──────────┘         │ (Relay)  │         └──────────┘      │
│                       └──────────┘                           │
│                                                               │
│  ✓ Works behind NAT/firewall                                 │
│  ✓ Auto-fallback, no manual config                           │
│  ✓ ~100-300ms latency per chunk                              │
└──────────────────────────────────────────────────────────────┘
```

### Connection Flow

1. **Download starts** → Worker tests direct TCP to first peer (5s timeout)
2. **If direct works** → Continue using TCP for all chunks
3. **If direct fails** → Switch to relay-only mode immediately
4. **Relay mode** → All chunks go through WebSocket relay
5. **No retry** → Once in relay mode, stay in relay mode

## 5. Luồng Hoạt Động

### 5.1 Share File (Seeder)

```
Peer                           Tracker
 │                                │
 │ 1. Chunk file (256KB each)     │
 │ 2. Calculate SHA-256 hashes    │
 │ 3. Build Merkle tree           │
 │                                │
 │ ───── POST /api/files/announce ─────▶
 │       {file_hash, chunks, size}│
 │                                │
 │ ◀──── 200 OK ──────────────────│
 │       File registered          │
 │                                │
```

### 5.2 Download File (Leecher)

```
Leecher                        Tracker                       Seeder
   │                              │                              │
   │ ── GET /api/files/{hash} ───▶│                              │
   │ ◀── {metadata, chunk_hashes} │                              │
   │                              │                              │
   │ ── GET /files/{hash}/peers ─▶│                              │
   │ ◀── [{peer_id, addr}]        │                              │
   │                              │                              │
   │                              │                              │
   │ ═══════════ TRY DIRECT TCP ═══════════════════════════════▶│
   │                              │                         [OK?]│
   │ ◀═══════════ CHUNK DATA ═══════════════════════════════════│
   │                              │                              │
   │                         [If failed]                         │
   │                              │                              │
   │ ═══════════ TRY HOLE PUNCH ══════════════════════════════▶ │
   │ ◀══════════════════════════════════════════════════════════│
   │                              │                              │
   │                         [If failed]                         │
   │                              │                              │
   │ ─── WebSocket /relay ───────▶│                              │
   │                              │◀─── WebSocket /relay ────────│
   │ ◀════════ RELAY DATA ════════│═══════════════════════════▶ │
   │                              │                              │
   │ Verify chunk hash (SHA-256)  │                              │
   │ Verify Merkle proof          │                              │
   │ Assemble file                │                              │
   │                              │                              │
```

## 6. Data Models

### Peer

```go
type Peer struct {
    ID        string    // UUID
    Address   string    // IP:Port
    Status    string    // online/offline
    Files     []string  // File hashes
    LastSeen  time.Time
    Bandwidth int64     // Bytes/sec limit
}
```

### File

```go
type File struct {
    Hash       string      // SHA-256 of entire file
    Name       string      // Original filename
    Size       int64       // Total bytes
    ChunkSize  int         // Bytes per chunk (256KB)
    Chunks     []ChunkInfo // Hash per chunk
    MerkleRoot string      // Merkle tree root
    Seeders    []string    // Peer IDs
}
```

### Chunk

```go
type ChunkInfo struct {
    Index int
    Hash  string // SHA-256
    Size  int
}
```

## 7. Security

### 7.1 Authentication
- API Keys for tracker access
- Middleware validates `X-API-Key` header

### 7.2 End-to-End Encryption
- **Key Exchange**: X25519 ECDH
- **Encryption**: AES-256-GCM
- **Key Derivation**: HKDF-SHA256

### 7.3 Integrity Verification
- SHA-256 hash per chunk
- Merkle tree for efficient verification
- Reject corrupted chunks

## 8. Công Nghệ Sử Dụng

| Thành phần | Công nghệ | Ghi chú |
|------------|-----------|---------|
| Ngôn ngữ | Go 1.21+ | Performance, concurrency |
| HTTP Framework | gorilla/mux | Routing, middleware |
| WebSocket | gorilla/websocket | Real-time, relay |
| Crypto | x/crypto | X25519, HKDF |
| Container | Docker | Containerization |
| Orchestration | Kubernetes | Scaling, deployment |
| Ingress | NGINX | TLS termination |

## 9. Deployment

### Development
```bash
make build
./bin/tracker &
./bin/peer -daemon -data ./data
```

### Production (Kubernetes)
```bash
kubectl apply -f k8s/
```

### Servers (Bare Metal)
```bash
scp bin/peer-linux-amd64 user@server:/opt/p2p/peer
systemctl start p2p-peer
```

