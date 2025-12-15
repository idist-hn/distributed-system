​​Câu 1:
Một công ty đang sử dụng một hệ thống quản lý kho (Warehouse Management System – WMS) cũ, trong đó giao diện API không tương thích với hệ thống thương mại điện tử mới của họ.
Hãy áp dụng khái niệm wrapper để đề xuất cách tích hợp hai hệ thống này.
Mô tả cách wrapper giúp giải quyết vấn đề tương thích giao diện và so sánh chi phí phát triển nếu sử dụng O(N²) wrappers trực tiếp giữa các hệ thống so với giải pháp message broker.

---

## Trả lời:

### 1. Đề xuất giải pháp sử dụng Wrapper

**Wrapper Pattern** là một design pattern cho phép chuyển đổi giao diện của một hệ thống thành giao diện mà hệ thống khác mong đợi, hoạt động như một lớp trung gian (adapter) giữa hai hệ thống không tương thích.

#### Kiến trúc đề xuất:

```
[E-commerce System] <---> [WMS Wrapper] <---> [Legacy WMS]
```

**Các thành phần:**

1. **WMS Wrapper Service**: Một service trung gian đóng vai trò adapter
   - Nhận request từ hệ thống e-commerce (format mới)
   - Chuyển đổi request sang format của WMS cũ
   - Gọi API của WMS cũ
   - Nhận response từ WMS cũ
   - Chuyển đổi response về format mà e-commerce hiểu được
   - Trả về cho hệ thống e-commerce

2. **Interface Mapping Layer**: Ánh xạ giữa các API endpoints
   ```
   E-commerce API          →  WMS Wrapper  →  Legacy WMS API
   POST /orders/inventory  →  Transform    →  GET /stock/check?sku=xxx
   GET /products/stock     →  Transform    →  POST /inventory/query
   ```

3. **Data Transformation Layer**: Chuyển đổi cấu trúc dữ liệu
   ```json
   // E-commerce format (input)
   {
     "productId": "PROD-123",
     "quantity": 10
   }

   // WMS format (output after transformation)
   {
     "sku": "PROD-123",
     "qty": 10,
     "warehouse_id": "WH-01"
   }
   ```

### 2. Cách Wrapper giải quyết vấn đề tương thích giao diện

**a) Protocol Adaptation:**
- WMS cũ có thể dùng SOAP, XML-RPC
- E-commerce mới dùng RESTful JSON
- Wrapper chuyển đổi giữa các protocol khác nhau

**b) Data Format Transformation:**
- Chuyển đổi cấu trúc dữ liệu (JSON ↔ XML)
- Mapping tên trường (productId → sku)
- Chuyển đổi kiểu dữ liệu (string ↔ integer)

**c) Business Logic Translation:**
- WMS cũ có thể yêu cầu nhiều API calls để hoàn thành một nghiệp vụ
- Wrapper có thể gộp nhiều calls thành một endpoint duy nhất cho e-commerce

**d) Error Handling:**
- Chuẩn hóa error codes và messages
- Retry logic cho các lỗi tạm thời
- Fallback mechanisms

**e) Security & Authentication:**
- E-commerce dùng OAuth 2.0/JWT
- WMS cũ dùng Basic Auth hoặc API Key
- Wrapper xử lý việc chuyển đổi authentication

### 3. So sánh chi phí xử lý: O(N²) Wrappers vs Message Broker

#### **Giải pháp 1: Direct Wrappers - O(N²)**

**Mô hình:**
```
Với N hệ thống, mỗi hệ thống cần wrapper riêng để giao tiếp với (N-1) hệ thống khác
→ Tổng số wrappers = N × (N-1) = O(N²)
→ Tổng số connections (kết nối) = N × (N-1) / 2 = O(N²)
```

**Ví dụ với 4 hệ thống:**
- E-commerce System
- WMS (Warehouse)
- CRM (Customer)
- ERP (Enterprise Resource Planning)

```
              ┌─────────────────────────────────────────┐
              │     DIRECT WRAPPERS (Point-to-Point)    │
              └─────────────────────────────────────────┘

       [E-commerce] ←──────────────────→ [WMS]
            ↑ ↘                        ↗ ↑
            │   ↘                    ↗   │
            │     ↘                ↗     │
            │       ↘            ↗       │
            ↓         ↘        ↗         ↓
         [CRM] ←───────────────────→ [ERP]

       Số connections: 4 × 3 / 2 = 6 bidirectional connections
       Số wrappers: 4 × 3 = 12 wrappers (mỗi chiều 1 wrapper)
```

**Chi phí xử lý (Processing Cost):**

| Metric | Giá trị | Công thức |
|--------|---------|-----------|
| **Số wrappers** | 12 | N × (N-1) = 4 × 3 |
| **Số connections** | 6 | N × (N-1) / 2 |
| **Transformations per message** | 1 | Direct A→B |
| **Hops per message** | 1 | Source → Destination |

**Phân tích chi phí xử lý:**

1. **Message Overhead per Request:**
   ```
   E-commerce → WMS:
   ┌────────────┐    ┌─────────────┐    ┌─────────┐
   │ E-commerce │ →  │ WMS Wrapper │ →  │   WMS   │
   └────────────┘    └─────────────┘    └─────────┘

   Processing: 1 transformation + 1 hop
   Latency: ~5-10ms (direct connection)
   ```

2. **Tổng số processing units khi scale:**
   ```
   N = 4:  4 × 3 = 12 wrappers
   N = 5:  5 × 4 = 20 wrappers (+67%)
   N = 6:  6 × 5 = 30 wrappers (+50%)
   N = 10: 10 × 9 = 90 wrappers
   N = 20: 20 × 19 = 380 wrappers
   ```

3. **Độ phức tạp khi thay đổi API:**
   ```
   Khi WMS thay đổi API:
   → Phải update 3 wrappers (từ E-commerce, CRM, ERP)
   → Chi phí xử lý: O(N-1) modifications
   ```

#### **Giải pháp 2: Message Broker - O(N)**

**Mô hình:**
```
Mỗi hệ thống chỉ cần 1 adapter để kết nối với Message Broker
→ Tổng số adapters = N = O(N)
→ Tổng số connections = N (mỗi system → broker)
```

**Kiến trúc:**
```
              ┌─────────────────────────────────────────┐
              │         MESSAGE BROKER (Hub-Spoke)       │
              └─────────────────────────────────────────┘

                      [E-commerce]
                           │ Adapter
                           ↓
       [WMS] ────→ ┌─────────────────┐ ←──── [CRM]
          Adapter  │  Message Broker │  Adapter
                   │  (RabbitMQ/     │
                   │   Kafka)        │
                   └────────┬────────┘
                            │
                            ↓ Adapter
                         [ERP]

       Số connections: 4 (mỗi system 1 connection)
       Số adapters: 4 (mỗi system 1 adapter)
```

**Chi phí xử lý (Processing Cost):**

| Metric | Giá trị | Công thức |
|--------|---------|-----------|
| **Số adapters** | 4 | N |
| **Số connections** | 4 | N |
| **Transformations per message** | 2 | Source adapter + Dest adapter |
| **Hops per message** | 2 | Source → Broker → Destination |

**Phân tích chi phí xử lý:**

1. **Message Overhead per Request:**
   ```
   E-commerce → WMS:
   ┌────────────┐    ┌─────────┐    ┌────────┐    ┌─────────┐    ┌─────────┐
   │ E-commerce │ →  │Adapter 1│ →  │ Broker │ →  │Adapter 2│ →  │   WMS   │
   └────────────┘    └─────────┘    └────────┘    └─────────┘    └─────────┘

   Processing: 2 transformations + 2 hops + queue operations
   Latency: ~10-50ms (qua broker)
   ```

2. **Tổng số processing units khi scale:**
   ```
   N = 4:  4 adapters + 1 broker = 5 components
   N = 5:  5 adapters + 1 broker = 6 components (+20%)
   N = 6:  6 adapters + 1 broker = 7 components (+17%)
   N = 10: 10 adapters + 1 broker = 11 components
   N = 20: 20 adapters + 1 broker = 21 components
   ```

3. **Độ phức tạp khi thay đổi API:**
   ```
   Khi WMS thay đổi API:
   → Chỉ cần update 1 adapter (WMS adapter)
   → Chi phí xử lý: O(1) modification
   ```

### 4. Bảng so sánh chi phí xử lý

| Tiêu chí | O(N²) Direct Wrappers | O(N) Message Broker |
|----------|:--------------------:|:-------------------:|
| **Số components (N=4)** | 12 wrappers | 4 adapters + 1 broker |
| **Số connections** | N×(N-1)/2 = 6 | N = 4 |
| **Hops per message** | 1 | 2 |
| **Transformations/msg** | 1 | 2 |
| **Latency per request** | ⭐ Thấp (5-10ms) | Cao hơn (10-50ms) |
| **Khi thêm 1 system** | +2N wrappers | +1 adapter |
| **Khi sửa 1 API** | O(N-1) updates | ⭐ O(1) update |
| **Scaling complexity** | O(N²) | ⭐ O(N) |
| **Bottleneck** | Không | Broker (cần cluster) |

### 5. Phân tích chi tiết theo N

```
CHI PHÍ XỬ LÝ THEO SỐ HỆ THỐNG:
─────────────────────────────────

N (systems) │ Direct Wrappers │ Message Broker │ Winner
────────────┼─────────────────┼────────────────┼──────────────
     2      │     2 wrappers  │   2 + broker   │ Direct
     3      │     6 wrappers  │   3 + broker   │ ~Same
     4      │    12 wrappers  │   4 + broker   │ Broker
     5      │    20 wrappers  │   5 + broker   │ Broker ✓
    10      │    90 wrappers  │  10 + broker   │ Broker ✓✓
    20      │   380 wrappers  │  20 + broker   │ Broker ✓✓✓

Break-even point: N ≈ 3-4 systems
```

```
BIỂU ĐỒ SO SÁNH:
────────────────

Components
    │
400 ┤                                          ╱ O(N²)
    │                                        ╱
300 ┤                                      ╱
    │                                    ╱
200 ┤                                  ╱
    │                                ╱
100 ┤                    ╱─────────╱
    │         ╱────────╱
 50 ┤    ╱───╱
    │  ╱╱     ─────────────────────────────── O(N)
    ├──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──→ N
    0  2  4  6  8  10 12 14 16 18 20
```

### 6. Trade-offs chi phí xử lý

| Aspect | Direct Wrappers | Message Broker |
|--------|-----------------|----------------|
| **Per-message latency** | ⭐ Thấp | Cao hơn (extra hop) |
| **Per-message throughput** | Cao | Phụ thuộc broker |
| **Total system complexity** | O(N²) | ⭐ O(N) |
| **API change impact** | O(N-1) | ⭐ O(1) |
| **New system integration** | O(2N) | ⭐ O(1) |
| **Single point of failure** | ⭐ Không | Có (broker) |
| **Message ordering** | Khó đảm bảo | ⭐ Dễ (queue) |
| **Async processing** | Khó | ⭐ Native support |

### 7. Kết luận và Khuyến nghị

**Về chi phí xử lý:**

| Số hệ thống | Khuyến nghị | Lý do |
|-------------|-------------|-------|
| N = 2 | Direct Wrapper | Đơn giản, latency thấp |
| N = 3-4 | Tùy yêu cầu | Break-even point |
| N ≥ 5 | Message Broker | O(N) << O(N²) |

**Cho bài toán hiện tại (2 hệ thống: E-commerce + WMS):**
- **Khuyến nghị: Direct Wrapper**
- Chi phí xử lý: 1 wrapper, 1 hop, latency thấp
- Đơn giản, không cần infrastructure phức tạp

**Khi mở rộng (≥5 hệ thống):**
- **Khuyến nghị: Message Broker**
- Chi phí xử lý giảm từ O(N²) xuống O(N)
- Trade-off: latency cao hơn (2 hops) nhưng scalability tốt hơn
- Khi API thay đổi: chỉ update O(1) adapter thay vì O(N-1) wrappers

