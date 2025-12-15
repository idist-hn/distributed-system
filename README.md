# P2P File Sharing System

Hệ thống chia sẻ file ngang hàng (Peer-to-Peer) được xây dựng bằng Go, với kiến trúc hybrid P2P sử dụng tracker để điều phối.

## 🌟 Highlights

- **Hybrid P2P Architecture**: Tracker điều phối, peers trao đổi file trực tiếp
- **Multi-Connection Strategy**: Direct TCP → UDP Hole Punch → WebSocket Relay
- **Advanced Features**: Parallel downloads, E2E encryption, DHT, Merkle verification
- **Production Ready**: Kubernetes deployment, Web dashboard, API authentication

## 📐 Kiến Trúc

```
┌─────────────────────────────────────────────────────────────────────┐
│                       TRACKER SERVER                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │  REST API    │  │  WebSocket   │  │   Dashboard  │              │
│  │  /api/*      │  │  Relay Hub   │  │   /dashboard │              │
│  └──────────────┘  └──────────────┘  └──────────────┘              │
│         │                 │                  │                      │
│         └─────────────────┼──────────────────┘                      │
│                           ▼                                         │
│  ┌────────────────────────────────────────────────────────────┐    │
│  │                    In-Memory Storage                        │    │
│  │  • Peer Registry  • File Metadata  • Relay Connections     │    │
│  └────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
                    │                    │
          ┌─────────┴─────────┐ ┌────────┴────────┐
          ▼                   ▼ ▼                  ▼
   ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
   │   PEER A     │◄──►│   PEER B     │◄──►│   PEER C     │
   │              │    │              │    │              │
   │ ┌──────────┐ │    │ ┌──────────┐ │    │ ┌──────────┐ │
   │ │P2P Server│ │    │ │P2P Server│ │    │ │P2P Server│ │
   │ │TCP:6881  │ │    │ │TCP:6882  │ │    │ │TCP:6883  │ │
   │ └──────────┘ │    │ └──────────┘ │    │ └──────────┘ │
   │ ┌──────────┐ │    │ ┌──────────┐ │    │ ┌──────────┐ │
   │ │ Storage  │ │    │ │ Storage  │ │    │ │ Storage  │ │
   │ │ ./data/  │ │    │ │ ./data/  │ │    │ │ ./data/  │ │
   │ └──────────┘ │    │ └──────────┘ │    │ └──────────┘ │
   └──────────────┘    └──────────────┘    └──────────────┘
```

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Docker (for Kubernetes deployment)
- kubectl (for K8s deployment)

### Local Development

```bash
# Clone repository
git clone https://github.com/your-org/distributed-system.git
cd distributed-system

# Build binaries
make build

# Start Tracker
./bin/tracker -addr :8080

# Start Peer (in another terminal)
./bin/peer -port 6881 -data ./data -tracker http://localhost:8080
```

### Docker Deployment

```bash
# Build images
docker build -f docker/tracker.Dockerfile -t p2p-tracker .
docker build -f docker/peer.Dockerfile -t p2p-peer .

# Run
docker run -p 8080:8080 p2p-tracker
docker run -p 6881:6881 p2p-peer
```

### Kubernetes Deployment

```bash
# Apply configurations
kubectl apply -f k8s/

# Check status
kubectl get pods -n p2p-system
```

## 📁 Project Structure

```
distributed-system/
├── bin/                          # Compiled binaries
│   ├── peer-linux-amd64
│   ├── peer-darwin-arm64
│   └── tracker
├── docker/                       # Dockerfiles
│   ├── peer.Dockerfile
│   └── tracker.Dockerfile
├── docs/                         # Documentation
│   ├── architecture.md
│   ├── protocol.md
│   └── features/                 # Feature documentation
├── k8s/                          # Kubernetes manifests
│   ├── tracker-deployment.yaml
│   ├── peer-statefulset.yaml
│   └── ingress.yaml
├── pkg/                          # Shared packages
│   ├── chunker/                  # File chunking (256KB)
│   ├── crypto/                   # E2E encryption (X25519, AES-GCM)
│   ├── dht/                      # Kademlia DHT
│   ├── hash/                     # SHA-256 hashing
│   ├── holepunch/                # UDP NAT hole punching
│   ├── merkle/                   # Merkle tree verification
│   ├── protocol/                 # Message definitions
│   └── throttle/                 # Bandwidth limiting
├── services/
│   ├── tracker/                  # Tracker Server
│   │   ├── cmd/                  # Entry point
│   │   └── internal/
│   │       ├── api/              # REST handlers, WebSocket, Dashboard
│   │       ├── models/           # Data models
│   │       └── storage/          # In-memory storage
│   └── peer/                     # Peer Node
│       ├── cmd/                  # Entry point
│       └── internal/
│           ├── client/           # Tracker client
│           ├── connection/       # Connection manager
│           ├── downloader/       # Parallel chunk downloader
│           ├── p2p/              # TCP P2P client/server
│           ├── relay/            # WebSocket relay client
│           └── storage/          # Local file storage
├── scripts/                      # Utility scripts
└── tests/                        # Test configurations
```

## 🔌 API Reference

### Tracker REST API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/dashboard` | Web UI Dashboard |
| POST | `/api/peers/register` | Register a peer |
| POST | `/api/peers/heartbeat` | Peer heartbeat |
| DELETE | `/api/peers/{id}` | Unregister peer |
| POST | `/api/files/announce` | Announce new file |
| GET | `/api/files` | List all files |
| GET | `/api/files/{hash}` | Get file metadata |
| GET | `/api/files/{hash}/peers` | Get peers for file |

### WebSocket Endpoints

| Endpoint | Description |
|----------|-------------|
| `/ws` | Real-time events (peer/file updates) |
| `/relay?peer_id=XXX` | Relay tunnel for NAT traversal |

## 💻 Peer CLI Commands

| Command | Description |
|---------|-------------|
| `share <path>` | Share a file |
| `list` | List available files |
| `download <hash>` | Download file by hash |
| `status` | Show peer status |
| `peers` | List connected peers |
| `quit` | Exit peer |

## ✨ Features

### Core Features
- [x] Tracker Server with REST API
- [x] Peer registration & heartbeat
- [x] File chunking (256KB chunks)
- [x] SHA-256 integrity verification
- [x] Auto-scan & share files in daemon mode

### Connection Strategy
- [x] Direct TCP connection
- [x] UDP NAT Hole Punching
- [x] WebSocket Relay (fallback)
- [x] Connection Manager (auto-fallback)

### Advanced Features
- [x] Parallel chunk downloads
- [x] Resume/Pause downloads
- [x] End-to-End encryption (X25519 + AES-256-GCM)
- [x] Kademlia DHT for peer discovery
- [x] Web UI Dashboard
- [x] Bandwidth throttling
- [x] Merkle tree verification

## 📊 Web Dashboard

Access the dashboard at `https://your-tracker/dashboard`:

- **Real-time Stats**: Peers online, files shared, relay connections
- **Peer List**: All connected peers with status
- **File List**: All shared files with metadata
- **Auto-refresh**: Updates every 30 seconds

## 🧪 Testing

```bash
# Run all tests
make test

# Run with coverage
go test -cover ./...

# Run specific package tests
go test -v ./pkg/merkle/...
go test -v ./pkg/throttle/...
go test -v ./pkg/holepunch/...
```

## 📚 Documentation

- [Architecture](docs/architecture.md) - System design
- [Protocol](docs/protocol.md) - P2P protocol specification
- [Features](docs/features/) - Detailed feature docs
  - [Parallel Downloads](docs/features/parallel-chunk-downloads.md)
  - [Resume/Pause](docs/features/resume-pause-downloads.md)
  - [E2E Encryption](docs/features/end-to-end-encryption.md)
  - [DHT Kademlia](docs/features/dht-kademlia.md)
  - [NAT Hole Punching](docs/features/nat-hole-punching.md)
  - [Web Dashboard](docs/features/web-ui-dashboard.md)
  - [Bandwidth Throttling](docs/features/bandwidth-throttling.md)
  - [Merkle Verification](docs/features/merkle-tree-verification.md)

## 🛠️ Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `TRACKER_ADDR` | `:8080` | Tracker listen address |
| `API_KEYS` | - | Comma-separated API keys |
| `PEER_PORT` | `6881` | Peer P2P port |
| `DATA_DIR` | `./data` | Data storage directory |
| `BANDWIDTH_LIMIT` | `0` | Bandwidth limit (0=unlimited) |

## 📄 License

MIT

