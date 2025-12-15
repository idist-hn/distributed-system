# Câu 2: Kiến trúc 3 tầng (Three-Tiered Architecture)

## Đề bài:

Một hệ thống bán hàng trực tuyến được thiết kế theo kiến trúc 3 tầng (three-tiered architecture) gồm:
- **Client (UI layer)**: giao diện web cho khách hàng đặt hàng.
- **Application server (Processing layer)**: xử lý đơn hàng, tính toán khuyến mãi.
- **Database server (Data layer)**: lưu trữ sản phẩm và giao dịch.

Hãy mô tả:
1. Luồng xử lý khi khách hàng đặt một đơn hàng mới.
2. Ưu điểm của việc tách application server ra thành một tầng riêng thay vì để toàn bộ xử lý ở client hoặc database server.

---

## Trả lời:

### 1. Sơ đồ kiến trúc 3 tầng

```
┌─────────────────────────────────────────────────────────────────────┐
│                        THREE-TIERED ARCHITECTURE                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────┐                                                   │
│  │   Browser    │  Tier 1: CLIENT (UI Layer)                        │
│  │   (Web UI)   │  - Hiển thị giao diện                             │
│  │              │  - Nhập liệu từ người dùng                        │
│  └──────┬───────┘  - Validation cơ bản                              │
│         │                                                            │
│         │ HTTP Request/Response (JSON/HTML)                         │
│         ▼                                                            │
│  ┌──────────────┐                                                   │
│  │ Application  │  Tier 2: APPLICATION SERVER (Processing Layer)   │
│  │   Server     │  - Xử lý business logic                           │
│  │              │  - Tính toán khuyến mãi                           │
│  │  (Node.js/   │  - Validation nghiệp vụ                           │
│  │   Java/      │  - Session management                             │
│  │   Python)    │  - Authentication/Authorization                   │
│  └──────┬───────┘                                                   │
│         │                                                            │
│         │ SQL/ORM Queries                                           │
│         ▼                                                            │
│  ┌──────────────┐                                                   │
│  │  Database    │  Tier 3: DATABASE SERVER (Data Layer)            │
│  │   Server     │  - Lưu trữ dữ liệu                                │
│  │              │  - ACID transactions                              │
│  │  (MySQL/     │  - Data integrity                                 │
│  │   PostgreSQL)│  - Backup & Recovery                              │
│  └──────────────┘                                                   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 2. Luồng xử lý khi khách hàng đặt đơn hàng mới

```
┌─────────┐         ┌─────────────┐         ┌──────────┐
│ Client  │         │ App Server  │         │ Database │
└────┬────┘         └──────┬──────┘         └────┬─────┘
     │                     │                     │
     │ (1) Chọn sản phẩm   │                     │
     │     & Nhấn "Đặt hàng"                     │
     │─────────────────────>                     │
     │                     │                     │
     │                     │ (2) Query thông tin │
     │                     │     sản phẩm, tồn kho
     │                     │─────────────────────>
     │                     │                     │
     │                     │ (3) Return data     │
     │                     │<─────────────────────
     │                     │                     │
     │                     │ (4) Tính giá, áp dụng
     │                     │     khuyến mãi      │
     │                     │ ┌─────────────────┐ │
     │                     │ │ Business Logic  │ │
     │                     │ │ - Check tồn kho │ │
     │                     │ │ - Tính discount │ │
     │                     │ │ - Tính phí ship │ │
     │                     │ └─────────────────┘ │
     │                     │                     │
     │                     │ (5) INSERT đơn hàng │
     │                     │─────────────────────>
     │                     │                     │
     │                     │ (6) UPDATE tồn kho  │
     │                     │─────────────────────>
     │                     │                     │
     │                     │ (7) Confirm success │
     │                     │<─────────────────────
     │                     │                     │
     │ (8) Hiển thị xác nhận                     │
     │<─────────────────────                     │
     │     đơn hàng        │                     │
     │                     │                     │
```

#### Chi tiết các bước:

| Bước | Tầng | Mô tả |
|------|------|-------|
| **(1)** | Client → App Server | Khách hàng điền thông tin đơn hàng (sản phẩm, số lượng, địa chỉ) và nhấn "Đặt hàng". Browser gửi HTTP POST request đến App Server |
| **(2)** | App Server → Database | App Server query thông tin sản phẩm, giá, số lượng tồn kho |
| **(3)** | Database → App Server | Database trả về dữ liệu sản phẩm và inventory |
| **(4)** | App Server (internal) | **Xử lý business logic**: Kiểm tra tồn kho đủ không, tính tổng tiền, áp dụng mã khuyến mãi (giảm 10%, free ship), tính phí vận chuyển |
| **(5)** | App Server → Database | INSERT đơn hàng mới vào bảng `orders` và `order_items` |
| **(6)** | App Server → Database | UPDATE giảm số lượng tồn kho trong bảng `inventory` |
| **(7)** | Database → App Server | Xác nhận transaction thành công |
| **(8)** | App Server → Client | Trả về trang xác nhận đơn hàng với mã đơn, tổng tiền, thời gian giao hàng dự kiến |

### 3. Ưu điểm của việc tách Application Server thành tầng riêng

#### So sánh với phương án để xử lý ở Client:

| Tiêu chí | Xử lý ở Client | Tách App Server riêng |
|----------|----------------|----------------------|
| **Bảo mật** | Kém - logic lộ ra ngoài, dễ bị hack giá, fake discount | Tốt - logic ẩn phía server, client không can thiệp được |
| **Validation** | Dễ bypass - user có thể sửa JavaScript | Đáng tin cậy - server kiểm tra lại mọi thứ |
| **Hiệu năng client** | Nặng - browser phải xử lý nhiều | Nhẹ - chỉ render UI |
| **Cập nhật logic** | Khó - phải chờ user refresh/clear cache | Dễ - deploy server là xong |
| **Đa nền tảng** | Duplicate code cho web, mobile, API | Một backend phục vụ tất cả |

#### So sánh với phương án để xử lý ở Database (Stored Procedures):

| Tiêu chí | Xử lý ở Database | Tách App Server riêng |
|----------|------------------|----------------------|
| **Khả năng mở rộng** | Khó - database là bottleneck | Dễ - scale horizontal nhiều app servers |
| **Bảo trì code** | Khó - SQL/PL khó test, debug | Dễ - dùng ngôn ngữ phổ biến, có IDE hỗ trợ |
| **Tích hợp bên ngoài** | Rất khó - gọi API từ stored proc phức tạp | Dễ - call REST API, message queue |
| **Vendor lock-in** | Cao - phụ thuộc DB vendor | Thấp - có thể đổi database |
| **Tài nguyên DB** | Lãng phí CPU cho logic | DB tập trung vào I/O, query |

#### Tổng hợp ưu điểm của kiến trúc 3 tầng:

```
┌─────────────────────────────────────────────────────────────────┐
│                    ƯU ĐIỂM KIẾN TRÚC 3 TẦNG                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. SEPARATION OF CONCERNS (Phân tách trách nhiệm)              │
│     - UI: chỉ lo hiển thị                                       │
│     - App Server: chỉ lo business logic                         │
│     - Database: chỉ lo lưu trữ                                  │
│                                                                  │
│  2. SCALABILITY (Khả năng mở rộng)                              │
│     - Scale từng tầng độc lập                                   │
│     - Thêm app server khi traffic cao                           │
│     - Thêm read replica cho database                            │
│                                                                  │
│  3. MAINTAINABILITY (Dễ bảo trì)                                │
│     - Sửa UI không ảnh hưởng backend                            │
│     - Đổi database không cần sửa UI                             │
│     - Team chuyên biệt cho từng tầng                            │
│                                                                  │
│  4. SECURITY (Bảo mật)                                          │
│     - Database không expose trực tiếp ra internet               │
│     - Business logic được bảo vệ phía server                    │
│     - Dễ implement authentication/authorization                 │
│                                                                  │
│  5. REUSABILITY (Tái sử dụng)                                   │
│     - Một App Server phục vụ: Web, Mobile App, API              │
│     - Shared business logic across platforms                    │
│                                                                  │
│  6. TECHNOLOGY FLEXIBILITY                                       │
│     - Mỗi tầng có thể dùng công nghệ phù hợp nhất               │
│     - Frontend: React/Vue, Backend: Java/Node, DB: PostgreSQL   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 4. Ví dụ thực tế với hệ thống bán hàng

```
Ví dụ: Khách hàng mua 2 áo với mã giảm giá "SALE20"

CLIENT (Browser):
┌─────────────────────────────────────┐
│  Giỏ hàng:                          │
│  - Áo thun x2: hiển thị 200.000đ    │
│  - Mã giảm giá: [SALE20]            │
│  - [Đặt hàng]                       │
└─────────────────────────────────────┘
                    │
                    ▼ POST /api/orders
APPLICATION SERVER:
┌─────────────────────────────────────┐
│  1. Validate mã SALE20 còn hạn?     │
│  2. Check tồn kho >= 2?             │
│  3. Tính: 200.000 - 20% = 160.000đ  │
│  4. Phí ship: 30.000đ               │
│  5. Tổng: 190.000đ                  │
│  6. Tạo đơn hàng                    │
└─────────────────────────────────────┘
                    │
                    ▼ SQL Transaction
DATABASE:
┌─────────────────────────────────────┐
│  BEGIN TRANSACTION;                 │
│  INSERT INTO orders (...);          │
│  INSERT INTO order_items (...);     │
│  UPDATE inventory SET qty = qty - 2;│
│  UPDATE promotions SET used = +1;   │
│  COMMIT;                            │
└─────────────────────────────────────┘
```

### 5. Kết luận

Kiến trúc 3 tầng là pattern phổ biến và được chứng minh hiệu quả cho các hệ thống web, đặc biệt là e-commerce. Việc tách **Application Server** thành tầng riêng mang lại:
- **Bảo mật**: Logic nghiệp vụ được bảo vệ
- **Khả năng mở rộng**: Scale horizontal dễ dàng
- **Bảo trì**: Code dễ đọc, test, và deploy
- **Linh hoạt**: Phục vụ nhiều loại client (web, mobile, API)