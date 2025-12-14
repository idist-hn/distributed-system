# Kiến Trúc Hệ Thống P2P File Sharing

## 1. Tổng Quan

Hệ thống chia sẻ file ngang hàng (P2P) cho phép các peer trao đổi file trực tiếp với nhau mà không cần server trung gian lưu trữ file.

### Mô Hình: Hybrid P2P (với Tracker)

```
┌─────────────────────────────────────────────────────────────┐
│                    TRACKER SERVER                           │
│  - Quản lý danh sách peers                                  │
│  - Quản lý metadata files (tên, hash, size, chunks)         │
│  - Không lưu file thực tế                                   │
└─────────────────────────────────────────────────────────────┘
           │                    │                    │
           ▼                    ▼                    ▼
    ┌──────────┐         ┌──────────┐         ┌──────────┐
    │  PEER A  │◄───────►│  PEER B  │◄───────►│  PEER C  │
    │ (Seeder) │         │(Leecher) │         │ (Seeder) │
    └──────────┘         └──────────┘         └──────────┘
         ▲                                          │
         └──────────────────────────────────────────┘
                    Trao đổi file trực tiếp
```

## 2. Các Thành Phần

### 2.1 Tracker Server
- **Vai trò**: Điều phối, không lưu file
- **Chức năng**:
  - Quản lý registry của peers (online/offline)
  - Lưu metadata của files đang được chia sẻ
  - Cung cấp danh sách peers có file cần tải

### 2.2 Peer Node
- **Seeder**: Peer có file hoàn chỉnh, chia sẻ cho người khác
- **Leecher**: Peer đang tải file
- **Chức năng**:
  - Kết nối và đăng ký với Tracker
  - Chia file thành chunks, tính hash
  - Upload/Download chunks với các peers khác
  - Verify tính toàn vẹn của chunks

## 3. Luồng Hoạt Động

### 3.1 Upload (Chia sẻ file mới)
1. Peer chia file thành chunks (256KB - 1MB mỗi chunk)
2. Tính SHA-256 hash cho mỗi chunk và toàn bộ file
3. Gửi metadata (file info + chunk hashes) lên Tracker
4. Tracker lưu metadata và đánh dấu peer là seeder

### 3.2 Download (Tải file)
1. Peer query Tracker để lấy file metadata
2. Tracker trả về danh sách peers có file
3. Peer kết nối trực tiếp với các seeders
4. Download từng chunk từ các peers (có thể song song)
5. Verify hash của mỗi chunk
6. Ghép các chunks thành file hoàn chỉnh
7. Thông báo Tracker rằng mình cũng là seeder

## 4. Công Nghệ Sử Dụng

| Thành phần | Công nghệ | Ghi chú |
|------------|-----------|---------|
| Ngôn ngữ | Go / Python | Go cho hiệu năng, Python cho đơn giản |
| Tracker API | gRPC / REST | gRPC hiệu quả, REST dễ debug |
| Peer-to-Peer | TCP Socket | Truyền file ổn định |
| Database | SQLite | Đơn giản, không cần setup |
| Hash | SHA-256 | Kiểm tra tính toàn vẹn |
| Serialization | Protocol Buffers / JSON | Định dạng message |

## 5. Cấu Trúc Thư Mục Dự Án

```
distributed-system/
├── docs/                    # Tài liệu
├── proto/                   # Protocol Buffers definitions
├── services/
│   ├── tracker/            # Tracker Server
│   │   ├── cmd/            # Entry point
│   │   ├── internal/       # Business logic
│   │   └── api/            # API handlers
│   └── peer/               # Peer Node
│       ├── cmd/            # Entry point
│       ├── internal/       # Business logic
│       └── p2p/            # P2P communication
├── pkg/                    # Shared packages
│   ├── protocol/           # Message definitions
│   ├── chunker/            # File chunking logic
│   └── hash/               # Hashing utilities
└── scripts/                # Helper scripts
```

