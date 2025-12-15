# P2P File Sharing System
## Bài Tập Cuối Kỳ - Hệ Thống Phân Tán

---

# Nội Dung

1. Giới thiệu
2. Kiến trúc hệ thống
3. Các thành phần chính
4. Thuật toán
5. Demo
6. Kết luận

---

# 1. Giới Thiệu

## Mục tiêu
- Xây dựng hệ thống chia sẻ file ngang hàng (P2P)
- Không cần server trung gian lưu file
- Tải song song từ nhiều nguồn

## Công nghệ
- **Ngôn ngữ**: Go (Golang)
- **Tracker API**: REST/HTTP
- **P2P Protocol**: TCP + JSON

---

# 2. Kiến Trúc Hệ Thống

```
┌─────────────────────────────────────┐
│          TRACKER SERVER             │
│  - Quản lý peers                    │
│  - Lưu metadata files               │
└─────────────────────────────────────┘
         │           │           │
         ▼           ▼           ▼
    ┌────────┐  ┌────────┐  ┌────────┐
    │ PEER A │◄►│ PEER B │◄►│ PEER C │
    │(Seeder)│  │(Leech) │  │(Seeder)│
    └────────┘  └────────┘  └────────┘
```

**Hybrid P2P**: Tracker quản lý, peers truyền file trực tiếp

---

# 3. Các Thành Phần

## Tracker Server
- REST API endpoints
- Quản lý peer online/offline
- Lưu file metadata (không lưu file)

## Peer Node
- CLI interface
- TCP server cho P2P
- File chunking & assembly
- Parallel download

---

# 4. Thuật Toán

## File Chunking
- Chia file thành chunks 256KB
- Mỗi chunk có hash SHA-256
- Verify integrity khi tải

## Parallel Download
1. Query Tracker → danh sách peers
2. Tạo worker pool (4 workers)
3. Mỗi worker tải 1 chunk
4. Verify hash → lưu disk
5. Assemble khi hoàn thành

---

# 5. Demo

## Bước 1: Khởi động Tracker
```bash
make run-tracker
```

## Bước 2: Peer 1 (Seeder)
```bash
./bin/peer -port 6881 -data ./data1
> share /path/to/file.zip
```

## Bước 3: Peer 2 (Leecher)
```bash
./bin/peer -port 6882 -data ./data2
> list
> download <file_hash>
```

---

# 6. Kết Quả

## Tính năng đã implement
✅ Tracker Server với REST API
✅ Peer registration & heartbeat
✅ File chunking (256KB)
✅ SHA-256 integrity verification
✅ Parallel download từ nhiều peers
✅ Bitfield exchange
✅ CLI interface

## Test coverage
- Unit tests: chunker, hash, storage
- Integration tests: full workflow

---

# 7. Hướng Phát Triển

1. **DHT**: Loại bỏ Tracker, hoàn toàn phân tán
2. **NAT Traversal**: Hỗ trợ peers sau NAT
3. **Encryption**: Mã hóa dữ liệu
4. **Web UI**: Giao diện web
5. **Mobile App**: Ứng dụng di động

---

# Kết Luận

- Đã xây dựng thành công hệ thống P2P file sharing
- Kiến trúc Hybrid P2P với Tracker
- Go + goroutines xử lý concurrent hiệu quả
- Sẵn sàng mở rộng với DHT

---

# Q&A

## Cảm ơn đã lắng nghe!

**Repository**: github.com/p2p-filesharing/distributed-system

**Liên hệ**: [email]

