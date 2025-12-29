# CLAUDE.md - Project Rules

## Project Overview

P2P File Sharing System - Hệ thống chia sẻ file ngang hàng với kiến trúc hybrid P2P.

## Architecture

### Components
- **Tracker Server** (`services/tracker/`): REST API, WebSocket relay, Dashboard
- **Peer Node** (`services/peer/`): P2P client/server, downloader, relay client
- **Shared Packages** (`pkg/`): chunker, hash, protocol, crypto

### Storage
- **Tracker**: PostgreSQL (production) / Memory (development)
- **Peer**: Local filesystem (`./data/shared/`, `./data/downloads/`)

## Key Design Decisions

### Connection Strategy
1. Try Direct TCP with **5 second timeout**
2. If fails → Switch to **relay-only mode** immediately
3. No retry direct TCP after switching to relay

### Chunk Size
- Default: **256KB** per chunk
- Configurable in `pkg/chunker/chunker.go`

### API Authentication
- API Key via `X-API-Key` header
- Keys configured via `API_KEYS` environment variable

## Code Conventions

### Go Style
- Use `log.Printf` for logging with prefixes: `[Tracker]`, `[Peer]`, `[Relay]`
- Error handling: return errors, don't panic
- Context: use for cancellation and timeouts

### File Structure
```
services/{service}/
├── cmd/           # Entry points
└── internal/      # Private packages
    ├── api/       # HTTP handlers
    ├── storage/   # Data storage
    └── ...
```

## Build Commands

```bash
make build              # Build all
make build-peer         # Build peer for current OS
make build-peer-linux   # Build peer for Linux
make build-download-linux  # Build download tool
./scripts/build-push.sh # Build and push Docker images
```

## Deployment

### Kubernetes
```bash
kubectl apply -f k8s/
kubectl rollout restart deployment/tracker -n p2p-system
```

### Bare Metal
```bash
scp bin/peer-linux-amd64 user@server:/opt/p2p/peer
systemctl restart p2p-peer
```

## Testing

```bash
go test ./...
go test -v ./pkg/chunker/...
```

## Important Files

| File                                              | Purpose                      |
| ------------------------------------------------- | ---------------------------- |
| `services/tracker/internal/api/router.go`         | Main tracker server          |
| `services/tracker/internal/api/relay.go`          | WebSocket relay hub          |
| `services/peer/internal/downloader/downloader.go` | Parallel download logic      |
| `services/peer/internal/relay/client.go`          | Relay client                 |
| `pkg/protocol/messages.go`                        | Protocol message definitions |

## Environment Variables

### Tracker
- `POSTGRES_URL`: PostgreSQL connection string
- `API_KEYS`: Comma-separated API keys
- `JWT_SECRET`: JWT signing secret

### Peer
- `-tracker`: Tracker URL
- `-api-key`: API key
- `-port`: P2P listen port
- `-data`: Data directory

