# Web UI Dashboard

## Tổng quan

**Web UI Dashboard** cung cấp giao diện trực quan để quản lý và giám sát hệ thống P2P tracker.

## Truy cập

```
https://p2p.idist.dev/dashboard
```

Hoặc truy cập root `/` sẽ redirect tự động đến dashboard.

## Giao diện

```
┌─────────────────────────────────────────────────────────────────┐
│  P2P Tracker Dashboard                        v1.2.0            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐           │
│  │ Peers    │ │ Files    │ │ Relay    │ │ Status   │           │
│  │ Online   │ │ Shared   │ │ Conns    │ │ Healthy  │           │
│  │    5     │ │    12    │ │    3     │ │    ✓     │           │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘           │
│                                                                  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ Connected Peers                                           │  │
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
│  │ def456...  | document.pdf      | 5.2MiB | 1     | 2024-01 │  │
│  └───────────────────────────────────────────────────────────┘  │
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

## Auto-refresh

Dashboard hiển thị thời gian refresh cuối cùng. Để refresh:
- Reload trang (F5)
- Hoặc implement auto-refresh với JavaScript (tùy chọn)

## Code Structure

```
services/tracker/internal/api/
├── dashboard.go          # Dashboard handler
└── templates/
    └── dashboard.html    # HTML template
```

## Customization

### Thêm stats mới

```go
// Trong DashboardData struct
type DashboardData struct {
    // ... existing fields
    CustomMetric int
}

// Trong DashboardHandler
data := DashboardData{
    CustomMetric: getCustomMetric(),
}
```

### Thêm table mới

1. Thêm struct View trong dashboard.go
2. Thêm data vào DashboardData
3. Cập nhật template HTML

## Lưu ý

1. Dashboard sử dụng CDN cho TailwindCSS và Lucide Icons
2. Templates được embed vào binary (không cần files riêng khi deploy)
3. Auth middleware bỏ qua `/dashboard` để public access

