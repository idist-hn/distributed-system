# P2P File Sharing System

Hệ thống chia sẻ file ngang hàng (Peer-to-Peer) được xây dựng bằng Go, với kiến trúc hybrid P2P sử dụng tracker để điều phối.

## 🌟 Highlights

- **Hybrid P2P Architecture**: Tracker điều phối, peers trao đổi file trực tiếp
- **Smart Connection Strategy**: Direct TCP (5s timeout) → WebSocket Relay (auto-fallback)
- **NAT Traversal**: WebSocket Relay cho peers behind NAT/firewall
- **Parallel Downloads**: Multi-worker chunk downloads với peer scoring
- **Production Ready**: PostgreSQL storage, Kubernetes deployment, Web dashboard
- **Real-time Monitoring**: WebSocket events, Prometheus metrics, live dashboard

## 🔗 Live Demo

- **Tracker**: https://p2p.idist.dev
- **Dashboard**: https://p2p.idist.dev/dashboard
- **API Docs**: https://p2p.idist.dev/health

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
│  │                    PostgreSQL Storage                       │    │
│  │  • Peer Registry  • File Metadata  • Chunk Hashes          │    │
│  └────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
                    │                    │
          ┌─────────┴─────────┐ ┌────────┴────────┐
          ▼                   ▼ ▼                  ▼
   ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
   │   PEER A     │◄──►│   PEER B     │◄──►│   PEER C     │
   │  (Seeder)    │    │  (Leecher)   │    │  (Seeder)    │
   │ ┌──────────┐ │    │ ┌──────────┐ │    │ ┌──────────┐ │
   │ │P2P Server│ │    │ │Downloader│ │    │ │P2P Server│ │
   │ │TCP:6881  │ │    │ │ Parallel │ │    │ │TCP:6882  │ │
   │ └──────────┘ │    │ └──────────┘ │    │ └──────────┘ │
   │ ┌──────────┐ │    │ ┌──────────┐ │    │ ┌──────────┐ │
   │ │  Relay   │ │    │ │  Relay   │ │    │ │  Relay   │ │
   │ │  Client  │ │    │ │  Client  │ │    │ │  Client  │ │
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
│   │       └── storage/          # PostgreSQL + Memory storage
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

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/health` | Health check | No |
| GET | `/dashboard` | Web UI Dashboard | No |
| GET | `/metrics` | Prometheus metrics | No |
| POST | `/api/auth/login` | Get JWT token | API Key |
| POST | `/api/peers/register` | Register a peer | API Key |
| POST | `/api/peers/heartbeat` | Peer heartbeat | API Key |
| DELETE | `/api/peers/{id}` | Unregister peer | API Key |
| POST | `/api/files/announce` | Announce new file | API Key |
| GET | `/api/files` | List all files | API Key |
| GET | `/api/files/{hash}` | Get file metadata | API Key |
| GET | `/api/files/{hash}/peers` | Get peers for file | API Key |

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
- [x] Peer registration & heartbeat (30s interval)
- [x] File chunking (256KB chunks)
- [x] SHA-256 integrity verification
- [x] Auto-scan & share files in daemon mode
- [x] PostgreSQL persistent storage

### Smart Connection Strategy
- [x] Direct TCP connection (5s timeout)
- [x] WebSocket Relay fallback (auto-switch)
- [x] Relay-only mode for NAT/firewall peers
- [x] Connection pooling & reuse

### Download Features
- [x] Parallel chunk downloads (multi-worker)
- [x] Peer scoring & selection
- [x] Resume interrupted downloads
- [x] Progress tracking & statistics

### Monitoring & Security
- [x] Web UI Dashboard (real-time)
- [x] WebSocket live events
- [x] Prometheus metrics
- [x] API key authentication
- [x] Rate limiting

## 📊 Web Dashboard

Access the dashboard at `https://p2p.idist.dev/dashboard`:

- **Real-time Stats**: Peers online, files shared, relay connections
- **Peer List**: All connected peers with status, IP, port
- **File List**: All shared files with size, seeders count
- **WebSocket Events**: Live updates via `/ws` endpoint
- **Auto-refresh**: Updates every 5 seconds

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
- [Deployment](docs/deployment.md) - Deployment guide
- [Features](docs/features/) - Detailed feature docs
  - [Relay Connection](docs/features/relay-connection.md)
  - [Parallel Downloads](docs/features/parallel-chunk-downloads.md)
  - [Web Dashboard](docs/features/web-ui-dashboard.md)

## 🛠️ Configuration

### Tracker Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_URL` | - | PostgreSQL connection string |
| `API_KEYS` | - | Comma-separated API keys |
| `JWT_SECRET` | - | JWT signing secret |
| `RATE_LIMIT_RPS` | `100` | Requests per second limit |

### Peer CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-tracker` | `http://localhost:8080` | Tracker URL |
| `-port` | `6881` | P2P listen port |
| `-data` | `./data` | Data directory |
| `-api-key` | - | API key for tracker |
| `-daemon` | `false` | Run in daemon mode |

## 🚀 Quick Download

```bash
# Download a file by hash
p2p-download <file-hash>

# Example
p2p-download 1bbbdb80ca3c67027bb53a3b8550fe8290c2edbc19632c93e44f8b182dd147ae
```

## 📄 License

MIT

