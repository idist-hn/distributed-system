# Web UI Dashboard

## Tổng quan

**Web UI Dashboard** cung cấp giao diện trực quan để quản lý và giám sát hệ thống P2P tracker với **real-time updates qua WebSocket**.

## Truy cập

```
https://p2p.idist.dev/dashboard
```

Hoặc truy cập root `/` sẽ redirect tự động đến dashboard.

## Tính năng chính

| Feature | Mô tả |
|---------|-------|
| **Real-time Updates** | WebSocket connection tự động cập nhật data |
| **Live Indicator** | Hiển thị trạng thái kết nối WebSocket |
| **Toast Notifications** | Popup thông báo khi có events |
| **Auto-reconnect** | Tự động kết nối lại với exponential backoff |
| **Stats Cards** | Hiển thị metrics quan trọng |
| **Interactive Tables** | Peers và Files tables với live updates |

## Giao diện

```
┌─────────────────────────────────────────────────────────────────┐
│  P2P Tracker Dashboard                   🟢 Live     v1.3.0     │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐│
│  │ Peers    │ │ Files    │ │ Relay    │ │ WS       │ │ Status ││
│  │ Online   │ │ Shared   │ │ Conns    │ │ Clients  │ │ Healthy││
│  │    5     │ │    12    │ │    3     │ │    2     │ │   ✓    ││
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └────────┘│
│                                                                  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ Connected Peers                              [Live ●]     │  │
│  ├───────────────────────────────────────────────────────────┤  │
│  │ ID       | IP:Port          | Status | Files | Upload    │  │
│  │ abc12... | 192.168.1.1:6881 | Online | 5     | 1.2 GiB   │  │
│  │ def34... | 192.168.1.2:6881 | Online | 3     | 500 MiB   │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ Shared Files                                              │  │
│  ├───────────────────────────────────────────────────────────┤  │
│  │ Hash       | Name              | Size   | Peers | Added   │  │
│  │ abc123...  | movie.mp4         | 2.5GiB | 3     | 2024-01 │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌─────────────────────────────────────┐  ← Toast Notifications │
│  │ 🟢 Peer abc123 joined the network  │                        │
│  └─────────────────────────────────────┘                        │
└─────────────────────────────────────────────────────────────────┘
```

## Các thông tin hiển thị

### Stats Cards

| Card | Mô tả |
|------|-------|
| **Peers Online** | Số peers đang hoạt động |
| **Shared Files** | Tổng số files được chia sẻ |
| **Relay Connections** | Số kết nối relay đang active |
| **System Status** | Trạng thái hệ thống |

### Peers Table

| Column | Mô tả |
|--------|-------|
| Peer ID | ID của peer (truncated) |
| IP:Port | Địa chỉ kết nối |
| Status | Online/Offline |
| Last Seen | Thời gian heartbeat cuối |
| Files | Số files đang share |
| Upload | Tổng bytes đã upload |
| Download | Tổng bytes đã download |

### Files Table

| Column | Mô tả |
|--------|-------|
| Hash | Hash của file (truncated) |
| Name | Tên file |
| Size | Kích thước |
| Category | Phân loại file |
| Peers | Số peers có file |
| Added | Thời gian thêm |

## Tech Stack

- **Frontend**: TailwindCSS, Lucide Icons
- **Template**: Go html/template với embed.FS
- **Backend**: Go HTTP server

## API Endpoints liên quan

| Endpoint | Description |
|----------|-------------|
| `GET /dashboard` | Web UI Dashboard |
| `GET /health/detailed` | Chi tiết health check |
| `GET /metrics` | Prometheus metrics |
| `GET /api/admin/peers` | Danh sách peers (JSON) |
| `GET /api/files` | Danh sách files (JSON) |

## WebSocket Real-time

### Connection Flow

```
Browser                    Tracker
   │                          │
   │── GET /dashboard ───────▶│
   │◀── HTML + JS ────────────│
   │                          │
   │── WS /ws ───────────────▶│
   │◀── Connection OK ────────│
   │                          │
   │◀── stats_update (5s) ────│
   │◀── peer_joined ──────────│
   │◀── file_added ───────────│
   │                          │
```

### WebSocket Events

| Event | Trigger | Data |
|-------|---------|------|
| `stats_update` | Mỗi 5 giây | peers_online, files_count, relay_peers, ws_clients |
| `peer_joined` | Peer đăng ký | peer_id, ip, port, hostname |
| `peer_left` | Peer offline | peer_id |
| `file_added` | File được announce | hash, name, size, peer_id |
| `file_removed` | File bị xóa | hash |

### JavaScript Client

Dashboard sử dụng `DashboardWS` class với các tính năng:

```javascript
class DashboardWS {
    connect()           // Kết nối WebSocket
    onOpen()            // Handle connection open
    onClose()           // Handle disconnect, auto-reconnect
    onMessage(event)    // Parse và dispatch events
    handleEvent(event)  // Route to specific handlers
    onStatsUpdate(data) // Update stats cards
    onPeerJoined(data)  // Add row to peers table
    onPeerLeft(data)    // Remove row from peers table
    onFileAdded(data)   // Add row to files table
    showToast(msg, type)// Show notification
}
```

### Auto-reconnect

Khi mất kết nối, client tự động reconnect với exponential backoff:

| Attempt | Delay |
|---------|-------|
| 1 | 1 giây |
| 2 | 2 giây |
| 3 | 4 giây |
| 4 | 8 giây |
| 5+ | 30 giây (max) |

## Code Structure

```
services/tracker/internal/api/
├── dashboard.go          # Dashboard handler
├── websocket.go          # WebSocket hub & client
└── templates/
    └── dashboard.html    # HTML + JS template
```

## API Endpoints liên quan

| Endpoint | Description |
|----------|-------------|
| `GET /dashboard` | Web UI Dashboard |
| `WS /ws` | WebSocket endpoint (realtime) |
| `GET /health` | Health check |
| `GET /metrics` | Prometheus metrics |
| `GET /api/admin/peers` | Danh sách peers (JSON) |
| `GET /api/files` | Danh sách files (JSON) |

## Lưu ý

1. Dashboard sử dụng CDN cho TailwindCSS và Lucide Icons
2. Templates được embed vào binary (không cần files riêng khi deploy)
3. Auth middleware bỏ qua `/dashboard` và `/ws` để public access
4. WebSocket không cần API key authentication
5. Stats broadcast mỗi 5 giây từ server

