# P2P File Sharing System

Hệ thống chia sẻ file ngang hàng (Peer-to-Peer) được xây dựng bằng Go.

## Kiến Trúc

```
┌─────────────────────────────────────────────────────────────┐
│                    TRACKER SERVER                           │
│  - Quản lý danh sách peers                                  │
│  - Quản lý metadata files                                   │
│  - Không lưu file thực tế                                   │
└─────────────────────────────────────────────────────────────┘
           │                    │                    │
           ▼                    ▼                    ▼
    ┌──────────┐         ┌──────────┐         ┌──────────┐
    │  PEER A  │◄───────►│  PEER B  │◄───────►│  PEER C  │
    │ (Seeder) │         │(Leecher) │         │ (Seeder) │
    └──────────┘         └──────────┘         └──────────┘
```

## Cài Đặt

```bash
# Clone repository
git clone <repository-url>
cd distributed-system

# Build
make build

# Run tests
make test
```

## Sử Dụng

### 1. Khởi động Tracker Server

```bash
# Terminal 1
make run-tracker
# hoặc
./bin/tracker -addr :8080
```

### 2. Khởi động Peer Node (Seeder)

```bash
# Terminal 2
./bin/peer -port 6881 -data ./data1 -tracker http://localhost:8080

# Trong CLI của peer:
> share /path/to/your/file.zip
# Output: Hash: abc123def456...
```

### 3. Khởi động Peer Node (Leecher)

```bash
# Terminal 3
./bin/peer -port 6882 -data ./data2 -tracker http://localhost:8080

# Trong CLI của peer:
> list                    # Xem danh sách files
> download abc123def456   # Tải file theo hash
```

## API Endpoints

| Method | Endpoint | Mô tả |
|--------|----------|-------|
| GET | `/health` | Health check |
| POST | `/api/peers/register` | Đăng ký peer |
| POST | `/api/peers/heartbeat` | Heartbeat |
| DELETE | `/api/peers/{id}` | Peer rời mạng |
| POST | `/api/files/announce` | Chia sẻ file mới |
| GET | `/api/files` | Danh sách files |
| GET | `/api/files/{hash}/peers` | Peers có file |

## CLI Commands

| Command | Mô tả |
|---------|-------|
| `share <path>` | Chia sẻ file |
| `list` | Xem files có sẵn |
| `download <hash>` | Tải file |
| `status` | Trạng thái peer |
| `quit` | Thoát |

## Cấu Trúc Project

```
├── Makefile
├── pkg/
│   ├── chunker/     # File chunking
│   ├── hash/        # SHA-256 hashing
│   ├── logger/      # Logging
│   └── protocol/    # Message definitions
├── services/
│   ├── tracker/     # Tracker Server
│   └── peer/        # Peer Node
├── docs/
│   ├── architecture.md
│   └── protocol.md
└── scripts/
    └── demo.sh
```

## Tính Năng

- [x] Tracker Server với REST API
- [x] Peer registration và heartbeat
- [x] File chunking (256KB chunks)
- [x] SHA-256 integrity verification
- [x] Parallel download từ nhiều peers
- [x] Bitfield exchange
- [x] HAVE message broadcast

## Demo

```bash
# Chạy demo script
./scripts/demo.sh
```

## Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage
```

## License

MIT

