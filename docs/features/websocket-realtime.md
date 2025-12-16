# WebSocket Real-time Updates

## Tổng quan

Hệ thống sử dụng **WebSocket** để cung cấp real-time updates cho Dashboard và các clients khác. Server broadcast events khi có thay đổi trong hệ thống.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        TRACKER SERVER                           │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │   Handlers  │───▶│   WSHub     │───▶│  Broadcast  │         │
│  │ (API calls) │    │  (Manager)  │    │  (to all)   │         │
│  └─────────────┘    └─────────────┘    └─────────────┘         │
│                            │                                    │
│         ┌──────────────────┼──────────────────┐                │
│         ▼                  ▼                  ▼                │
│    ┌─────────┐        ┌─────────┐        ┌─────────┐          │
│    │ Client1 │        │ Client2 │        │ Client3 │          │
│    │ (WS)    │        │ (WS)    │        │ (WS)    │          │
│    └─────────┘        └─────────┘        └─────────┘          │
└─────────────────────────────────────────────────────────────────┘
         │                  │                  │
         ▼                  ▼                  ▼
    ┌─────────┐        ┌─────────┐        ┌─────────┐
    │Dashboard│        │Dashboard│        │  CLI    │
    │ Browser │        │ Browser │        │ Client  │
    └─────────┘        └─────────┘        └─────────┘
```

## Endpoint

```
wss://p2p.idist.dev/ws
```

**Note**: Endpoint `/ws` bypass authentication middleware.

## Event Types

### 1. `peer_joined`
Triggered khi peer đăng ký với tracker.

```json
{
  "type": "peer_joined",
  "data": {
    "peer_id": "peer-abc123",
    "ip": "192.168.1.100",
    "port": 6881,
    "hostname": "my-computer"
  },
  "timestamp": "2024-12-16T10:30:00Z"
}
```

### 2. `peer_left`
Triggered khi peer offline (heartbeat timeout).

```json
{
  "type": "peer_left",
  "data": {
    "peer_id": "peer-abc123"
  },
  "timestamp": "2024-12-16T10:35:00Z"
}
```

### 3. `file_added`
Triggered khi file mới được announce.

```json
{
  "type": "file_added",
  "data": {
    "hash": "abc123def456...",
    "name": "movie.mp4",
    "size": 1073741824,
    "peer_id": "peer-abc123"
  },
  "timestamp": "2024-12-16T10:31:00Z"
}
```

### 4. `file_removed`
Triggered khi file bị xóa.

```json
{
  "type": "file_removed",
  "data": {
    "hash": "abc123def456..."
  },
  "timestamp": "2024-12-16T10:40:00Z"
}
```

### 5. `stats_update`
Broadcast mỗi 5 giây với system stats.

```json
{
  "type": "stats_update",
  "data": {
    "peers_online": 15,
    "peers_total": 42,
    "files_count": 128,
    "relay_peers": 3,
    "ws_clients": 5
  },
  "timestamp": "2024-12-16T10:30:05Z"
}
```

## Server Implementation

### WSHub (WebSocket Hub)

```go
type WSHub struct {
    clients    map[*WSClient]bool
    broadcast  chan WSEvent
    register   chan *WSClient
    unregister chan *WSClient
    mu         sync.RWMutex
}
```

### Broadcasting Events

```go
// Trong handlers khi peer join
s.wsHub.Broadcast(WSEvent{
    Type: EventPeerJoined,
    Data: map[string]interface{}{
        "peer_id":  req.PeerID,
        "ip":       req.IP,
        "port":     req.Port,
        "hostname": req.Hostname,
    },
})
```

### Stats Broadcast Goroutine

```go
func (s *Server) StartStatsBroadcast(interval time.Duration) {
    go func() {
        ticker := time.NewTicker(interval)
        for range ticker.C {
            s.broadcastStats()
        }
    }()
}
```

## Client Implementation

### JavaScript Example

```javascript
class DashboardWS {
    constructor() {
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectDelay = 30000;
    }

    connect() {
        const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
        this.ws = new WebSocket(`${protocol}//${location.host}/ws`);
        
        this.ws.onopen = () => this.onOpen();
        this.ws.onclose = () => this.onClose();
        this.ws.onmessage = (e) => this.onMessage(e);
    }

    onMessage(event) {
        const msg = JSON.parse(event.data);
        this.handleEvent(msg);
    }

    handleEvent(event) {
        switch(event.type) {
            case 'peer_joined':  this.onPeerJoined(event.data); break;
            case 'peer_left':    this.onPeerLeft(event.data); break;
            case 'file_added':   this.onFileAdded(event.data); break;
            case 'stats_update': this.onStatsUpdate(event.data); break;
        }
    }
}
```

## Code Location

| File | Description |
|------|-------------|
| `services/tracker/internal/api/websocket.go` | WSHub, WSClient, ServeWS |
| `services/tracker/internal/api/router.go` | Route setup, StartStatsBroadcast |
| `services/tracker/internal/api/handlers.go` | Broadcast calls in handlers |
| `services/tracker/internal/api/templates/dashboard.html` | JS client |

## Configuration

| Setting | Value | Description |
|---------|-------|-------------|
| Stats Interval | 5 seconds | Thời gian giữa các stats broadcast |
| Ping Interval | 30 seconds | WebSocket ping để keep-alive |
| Pong Timeout | 60 seconds | Timeout nếu không nhận pong |
| Max Message Size | 512 bytes | Giới hạn message từ client |

## Lưu ý

1. WebSocket endpoint bypass auth middleware để dashboard public access
2. Client nên implement auto-reconnect với exponential backoff
3. Server broadcast đến tất cả connected clients
4. Sử dụng gorilla/websocket library

