# Báo Cáo Kỹ Thuật: Hệ Thống Chia Sẻ File P2P

## 1. Giới Thiệu

### 1.1 Mục Tiêu
Xây dựng hệ thống chia sẻ file ngang hàng (Peer-to-Peer) cho phép người dùng chia sẻ và tải file trực tiếp từ nhau mà không cần server trung gian lưu trữ file.

### 1.2 Phạm Vi
- Hybrid P2P với Tracker Server để quản lý metadata
- Truyền file trực tiếp giữa các peers
- Hỗ trợ tải song song từ nhiều nguồn

## 2. Kiến Trúc Hệ Thống

### 2.1 Tổng Quan
```
┌─────────────────────────────────────────────────────────────┐
│                    TRACKER SERVER                           │
│  - Quản lý danh sách peers (online/offline)                 │
│  - Lưu metadata files (tên, hash, chunks)                   │
│  - Cung cấp peer discovery                                  │
└─────────────────────────────────────────────────────────────┘
           │                    │                    │
           ▼                    ▼                    ▼
    ┌──────────┐         ┌──────────┐         ┌──────────┐
    │  PEER A  │◄───────►│  PEER B  │◄───────►│  PEER C  │
    │ (Seeder) │   TCP   │(Leecher) │   TCP   │ (Seeder) │
    └──────────┘         └──────────┘         └──────────┘
```

### 2.2 Thành Phần

| Thành phần | Chức năng | Công nghệ |
|------------|-----------|-----------|
| Tracker Server | Quản lý peers, files | Go, REST API |
| Peer Node | Chia sẻ/tải file | Go, TCP Socket |
| Chunker | Chia file thành chunks | Go |
| Hash | Xác thực integrity | SHA-256 |

## 3. Thuật Toán Chính

### 3.1 File Chunking
- Chunk size: 256KB (có thể cấu hình)
- Mỗi chunk có hash SHA-256 riêng
- File hash = SHA-256 của toàn bộ file

### 3.2 Parallel Download
```
1. Leecher query Tracker để lấy danh sách peers có file
2. Tạo worker pool (mặc định 4 workers)
3. Mỗi worker:
   a. Lấy chunk index từ queue
   b. Kết nối đến peer có chunk đó
   c. Tải chunk và verify hash
   d. Lưu chunk vào disk
4. Khi tất cả chunks hoàn thành, assemble file
```

### 3.3 Peer Discovery
```
1. Peer đăng ký với Tracker (IP, Port)
2. Gửi heartbeat mỗi 30 giây
3. Tracker đánh dấu offline nếu không nhận heartbeat > 60s
4. Khi share file: gửi metadata đến Tracker
5. Khi tìm file: query Tracker để lấy peers
```

## 4. Protocol

### 4.1 Tracker API (REST)
| Endpoint | Method | Mô tả |
|----------|--------|-------|
| `/api/peers/register` | POST | Đăng ký peer |
| `/api/peers/heartbeat` | POST | Heartbeat |
| `/api/files/announce` | POST | Chia sẻ file |
| `/api/files` | GET | Danh sách files |
| `/api/files/{hash}/peers` | GET | Peers có file |

### 4.2 P2P Protocol (TCP/JSON)
| Message | Mô tả |
|---------|-------|
| HANDSHAKE | Xác thực kết nối |
| REQUEST_CHUNK | Yêu cầu chunk |
| CHUNK_DATA | Dữ liệu chunk |
| BITFIELD | Chunks peer có |
| HAVE | Thông báo có chunk mới |

## 5. Cấu Trúc Code

```
distributed-system/
├── pkg/                    # Shared packages
│   ├── chunker/           # File chunking
│   ├── hash/              # SHA-256 hashing
│   ├── logger/            # Logging
│   └── protocol/          # Message definitions
├── services/
│   ├── tracker/           # Tracker Server
│   │   ├── cmd/           # Entry point
│   │   └── internal/
│   │       ├── api/       # REST handlers
│   │       ├── models/    # Data models
│   │       └── storage/   # In-memory storage
│   └── peer/              # Peer Node
│       ├── cmd/           # Entry point + CLI
│       └── internal/
│           ├── client/    # Tracker client
│           ├── downloader/# Download manager
│           ├── p2p/       # TCP server/client
│           └── storage/   # Local storage
└── docs/                  # Documentation
```

## 6. Kết Quả

### 6.1 Tính Năng Đã Implement
- [x] Tracker Server với REST API
- [x] Peer registration và heartbeat
- [x] File chunking (256KB)
- [x] SHA-256 integrity verification
- [x] Parallel download từ nhiều peers
- [x] Bitfield exchange
- [x] HAVE message broadcast
- [x] CLI interface

### 6.2 Test Coverage
- Unit tests: chunker, hash, storage, handlers
- Integration tests: full workflow

## 7. Hướng Phát Triển

1. **DHT (Distributed Hash Table)**: Loại bỏ Tracker, hoàn toàn phân tán
2. **NAT Traversal**: Hỗ trợ peers sau NAT
3. **Encryption**: Mã hóa dữ liệu truyền
4. **Web UI**: Giao diện web thay vì CLI
5. **Mobile App**: Ứng dụng di động

## 8. Kết Luận

Hệ thống P2P file sharing đã được xây dựng thành công với kiến trúc Hybrid P2P, cho phép chia sẻ và tải file hiệu quả giữa các peers. Việc sử dụng Go với goroutines giúp xử lý concurrent connections hiệu quả.

