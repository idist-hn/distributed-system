# 📋 Use Cases - P2P File Sharing System

## Tổng Quan

Tài liệu này mô tả chi tiết các Use Cases của hệ thống P2P File Sharing, bao gồm actors, preconditions, main flows, và alternative flows.

---

## Actors

| Actor              | Mô tả                                             |
| ------------------ | ------------------------------------------------- |
| **Peer**           | Node trong mạng P2P, có thể upload/download files |
| **Seeder**         | Peer có đầy đủ file và chỉ upload                 |
| **Leecher**        | Peer đang download file                           |
| **Tracker**        | Server điều phối, quản lý peers và files          |
| **Admin**          | Người quản trị hệ thống                           |
| **Dashboard User** | Người xem dashboard monitoring                    |

---

## UC-01: Đăng Ký Peer

### Thông tin cơ bản
| Thuộc tính      | Giá trị                                       |
| --------------- | --------------------------------------------- |
| **Use Case ID** | UC-01                                         |
| **Tên**         | Đăng Ký Peer                                  |
| **Actor**       | Peer                                          |
| **Mô tả**       | Peer đăng ký với Tracker để tham gia mạng P2P |

### Preconditions
- Tracker server đang hoạt động
- Peer có kết nối mạng
- Peer có API key hợp lệ

### Main Flow
1. Peer gửi request `POST /api/peers/register` với thông tin: peer_id, ip, port, hostname
2. Tracker validate API key
3. Tracker kiểm tra rate limit
4. Tracker lưu thông tin peer vào database
5. Tracker broadcast event `peer_joined` qua WebSocket
6. Tracker trả về response success

### Alternative Flows
- **A1**: API key không hợp lệ → Trả về 401 Unauthorized
- **A2**: Rate limit exceeded → Trả về 429 Too Many Requests
- **A3**: Peer đã tồn tại → Cập nhật thông tin peer

### Postconditions
- Peer được lưu trong database
- Dashboard hiển thị peer mới
- WebSocket clients nhận event peer_joined

---

## UC-02: Chia Sẻ File (Announce)

### Thông tin cơ bản
| Thuộc tính      | Giá trị                           |
| --------------- | --------------------------------- |
| **Use Case ID** | UC-02                             |
| **Tên**         | Chia Sẻ File                      |
| **Actor**       | Peer (Seeder)                     |
| **Mô tả**       | Peer thông báo có file để chia sẻ |

### Preconditions
- Peer đã đăng ký với Tracker
- File đã được chunk và hash

### Main Flow
1. Peer chunk file thành các phần 256KB
2. Peer tính SHA-256 hash cho mỗi chunk và toàn bộ file
3. Peer gửi request `POST /api/files/announce` với metadata
4. Tracker lưu file info và mapping peer-file
5. Tracker broadcast event `file_added` qua WebSocket
6. Tracker trả về file_id

### Alternative Flows
- **A1**: File đã tồn tại → Thêm peer vào danh sách seeders
- **A2**: Metadata không hợp lệ → Trả về 400 Bad Request

### Postconditions
- File metadata được lưu trong database
- Peer được liên kết với file
- File xuất hiện trên dashboard

---

## UC-03: Tìm Kiếm File

### Thông tin cơ bản
| Thuộc tính      | Giá trị                          |
| --------------- | -------------------------------- |
| **Use Case ID** | UC-03                            |
| **Tên**         | Tìm Kiếm File                    |
| **Actor**       | Peer (Leecher)                   |
| **Mô tả**       | Peer tìm kiếm file muốn download |

### Preconditions
- Peer đã đăng ký với Tracker

### Main Flow
1. Peer gửi request `GET /api/files/search?q=keyword`
2. Tracker thực hiện full-text search
3. Tracker trả về danh sách files matching

### Alternative Flows
- **A1**: Không tìm thấy → Trả về empty list
- **A2**: Search by hash → `GET /api/files/{hash}`
- **A3**: Parse magnet link → `GET /api/magnet?uri=magnet:...`

### Postconditions
- Peer có danh sách files có thể download

---

## UC-04: Download File

### Thông tin cơ bản
| Thuộc tính      | Giá trị                           |
| --------------- | --------------------------------- |
| **Use Case ID** | UC-04                             |
| **Tên**         | Download File                     |
| **Actor**       | Peer (Leecher)                    |
| **Mô tả**       | Peer download file từ các seeders |

### Preconditions
- Peer biết file hash
- Có ít nhất 1 seeder online

### Main Flow
1. Peer gửi `GET /api/files/{hash}/peers` để lấy danh sách seeders
2. Peer thử kết nối trực tiếp TCP đến seeder
3. Peer gửi HANDSHAKE message
4. Peer request chunks song song (parallel download)
5. Mỗi chunk nhận được → verify hash
6. Khi đủ chunks → assemble file
7. Verify hash toàn bộ file

### Alternative Flows
- **A1**: Direct TCP fail → Thử NAT Hole Punching
- **A2**: Hole punch fail → Sử dụng Relay qua Tracker
- **A3**: Chunk hash mismatch → Request lại từ peer khác
- **A4**: Peer disconnect → Resume từ peer khác
- **A5**: Endgame mode → Request chunk cuối từ nhiều peers

### Postconditions
- File được download hoàn chỉnh
- Hash verified
- Peer trở thành seeder cho file này

---

## UC-05: Heartbeat

### Thông tin cơ bản
| Thuộc tính      | Giá trị                                         |
| --------------- | ----------------------------------------------- |
| **Use Case ID** | UC-05                                           |
| **Tên**         | Heartbeat                                       |
| **Actor**       | Peer                                            |
| **Mô tả**       | Peer gửi heartbeat để duy trì trạng thái online |

### Preconditions
- Peer đã đăng ký

### Main Flow
1. Peer gửi `POST /api/peers/heartbeat` mỗi 30 giây
2. Tracker cập nhật last_seen
3. Tracker trả về next heartbeat interval

### Alternative Flows
- **A1**: Không heartbeat > 90s → Tracker đánh dấu offline
- **A2**: Peer offline → Broadcast event `peer_left`

### Postconditions
- Peer status được cập nhật

---

## UC-06: Xem Dashboard

### Thông tin cơ bản
| Thuộc tính      | Giá trị                          |
| --------------- | -------------------------------- |
| **Use Case ID** | UC-06                            |
| **Tên**         | Xem Dashboard                    |
| **Actor**       | Dashboard User                   |
| **Mô tả**       | Xem trạng thái hệ thống realtime |

### Preconditions
- Truy cập được URL dashboard

### Main Flow
1. User truy cập `/dashboard`
2. Browser load HTML với WebSocket client
3. WebSocket connect đến `/ws`
4. Dashboard hiển thị stats, peers, files
5. Khi có event → Dashboard tự động cập nhật
6. Toast notification hiển thị cho mỗi event

### Alternative Flows
- **A1**: WebSocket disconnect → Auto-reconnect với exponential backoff
- **A2**: Event peer_joined → Thêm row vào peers table
- **A3**: Event file_added → Thêm row vào files table
- **A4**: Event stats_update → Cập nhật stats cards

### Postconditions
- User thấy trạng thái realtime

---

## UC-07: NAT Traversal

### Thông tin cơ bản
| Thuộc tính      | Giá trị                            |
| --------------- | ---------------------------------- |
| **Use Case ID** | UC-07                              |
| **Tên**         | NAT Traversal                      |
| **Actor**       | Peer (behind NAT)                  |
| **Mô tả**       | Peer kết nối với peer khác qua NAT |

### Preconditions
- Cả 2 peers đều behind NAT
- Direct TCP connection fail

### Main Flow (Hole Punching)
1. Peer A request hole punch qua Tracker
2. Tracker gửi punch request đến Peer B
3. Cả 2 peers đồng thời gửi UDP packets
4. NAT mapping được tạo
5. Connection established

### Alternative Flow (Relay)
1. Hole punch fail sau 3 attempts
2. Peer A connect WebSocket `/relay`
3. Peer B connect WebSocket `/relay`
4. Tracker relay messages giữa 2 peers

### Postconditions
- Peers có thể communicate

---

## UC-08: Generate Magnet Link

### Thông tin cơ bản
| Thuộc tính      | Giá trị                       |
| --------------- | ----------------------------- |
| **Use Case ID** | UC-08                         |
| **Tên**         | Generate Magnet Link          |
| **Actor**       | Peer (Seeder)                 |
| **Mô tả**       | Tạo magnet link để share file |

### Main Flow
1. Peer gửi `GET /api/files/{hash}/magnet`
2. Tracker generate magnet URI với: hash, name, size, tracker URL
3. Trả về magnet link

### Magnet Format
```
magnet:?xt=urn:sha256:{hash}&dn={name}&xl={size}&tr={tracker_url}
```

### Postconditions
- User có magnet link để share

---

## UC-09: Admin Quản Lý

### Thông tin cơ bản
| Thuộc tính      | Giá trị               |
| --------------- | --------------------- |
| **Use Case ID** | UC-09                 |
| **Tên**         | Admin Quản Lý         |
| **Actor**       | Admin                 |
| **Mô tả**       | Quản trị hệ thống P2P |

### Main Flows
1. **Xem danh sách peers**: `GET /api/admin/peers`
2. **Kick peer**: `DELETE /api/admin/peers/{peer_id}`
3. **Xóa file**: `DELETE /api/admin/files/{hash}`
4. **Xem metrics**: `GET /metrics`
5. **Health check**: `GET /health`

### Preconditions
- Admin có API key với quyền admin

---

## UC-10: Resume Download

### Thông tin cơ bản
| Thuộc tính      | Giá trị                                |
| --------------- | -------------------------------------- |
| **Use Case ID** | UC-10                                  |
| **Tên**         | Resume Download                        |
| **Actor**       | Peer (Leecher)                         |
| **Mô tả**       | Tiếp tục download sau khi bị gián đoạn |

### Preconditions
- Download bị gián đoạn
- Progress file còn tồn tại

### Main Flow
1. Peer load progress từ `.progress` file
2. Peer xác định chunks đã download
3. Peer chỉ request các chunks còn thiếu
4. Continue download

### Postconditions
- Download tiếp tục không cần bắt đầu lại

---

## Use Case Diagram

```
                                    ┌─────────────────┐
                                    │     Tracker     │
                                    └────────┬────────┘
                                             │
        ┌────────────────────────────────────┼────────────────────────────────────┐
        │                                    │                                    │
        ▼                                    ▼                                    ▼
┌───────────────┐                   ┌───────────────┐                   ┌───────────────┐
│   UC-01       │                   │   UC-05       │                   │   UC-06       │
│ Register Peer │                   │  Heartbeat    │                   │  Dashboard    │
└───────────────┘                   └───────────────┘                   └───────────────┘
        │
        ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│   UC-02       │───▶│   UC-03       │───▶│   UC-04       │───▶│   UC-10       │
│ Announce File │    │ Search File   │    │ Download File │    │Resume Download│
└───────────────┘    └───────────────┘    └───────────────┘    └───────────────┘
        │                                         │
        ▼                                         ▼
┌───────────────┐                        ┌───────────────┐
│   UC-08       │                        │   UC-07       │
│ Magnet Link   │                        │ NAT Traversal │
└───────────────┘                        └───────────────┘

                    ┌───────────────┐
                    │   UC-09       │
                    │ Admin Manage  │◀──── Admin Actor
                    └───────────────┘
```

---

## Traceability Matrix

| Use Case | API Endpoints                        | Packages                           |
| -------- | ------------------------------------ | ---------------------------------- |
| UC-01    | POST /api/peers/register             | api/handlers.go                    |
| UC-02    | POST /api/files/announce             | api/handlers.go, pkg/chunker       |
| UC-03    | GET /api/files/search                | api/handlers.go                    |
| UC-04    | GET /api/files/{hash}/peers, TCP P2P | pkg/protocol, peer/p2p             |
| UC-05    | POST /api/peers/heartbeat            | api/handlers.go                    |
| UC-06    | GET /dashboard, WS /ws               | api/dashboard.go, api/websocket.go |
| UC-07    | WS /relay                            | pkg/holepunch, api/relay.go        |
| UC-08    | GET /api/files/{hash}/magnet         | pkg/magnet, api/handlers.go        |
| UC-09    | /api/admin/*                         | api/handlers.go                    |
| UC-10    | -                                    | peer/storage                       |

