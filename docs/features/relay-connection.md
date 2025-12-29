# WebSocket Relay Connection

## Overview

WebSocket Relay là tính năng cho phép peers behind NAT/firewall trao đổi dữ liệu thông qua tracker server.

## How It Works

```
┌──────────┐     WebSocket      ┌──────────┐     WebSocket      ┌──────────┐
│  Peer A  │◄──────────────────►│ Tracker  │◄──────────────────►│  Peer B  │
│  (NAT)   │                    │ (Relay)  │                    │  (NAT)   │
└──────────┘                    └──────────┘                    └──────────┘
     │                               │                               │
     │  1. Connect to /relay         │                               │
     │──────────────────────────────►│                               │
     │                               │◄──────────────────────────────│
     │                               │  2. Connect to /relay         │
     │                               │                               │
     │  3. Request chunk from Peer B │                               │
     │──────────────────────────────►│                               │
     │                               │  4. Forward request           │
     │                               │──────────────────────────────►│
     │                               │                               │
     │                               │◄──────────────────────────────│
     │                               │  5. Chunk response            │
     │◄──────────────────────────────│                               │
     │  6. Forward response          │                               │
```

## Smart Connection Strategy

1. **Test Direct TCP** (5 second timeout)
2. **If success** → Use direct TCP for all chunks
3. **If timeout** → Switch to relay-only mode immediately
4. **No retry** → Stay in relay mode for entire download

## Performance

| Method | Latency | Throughput |
|--------|---------|------------|
| Direct TCP | ~10ms | ~50 MB/s |
| WebSocket Relay | ~100-300ms | ~1-2 MB/s |

## Configuration

Relay is enabled by default. Peers auto-connect to relay on startup.

```go
// Peer connects to relay automatically
relayClient := relay.NewClient(trackerURL, peerID)
relayClient.Connect()
```

## Relay Hub (Server Side)

Tracker maintains a hub of connected peers:

```go
type RelayHub struct {
    peers map[string]*RelayPeer  // peer_id -> connection
    mu    sync.RWMutex
}
```

