Câu 4:

Giả sử bạn xây dựng một ứng dụng chia sẻ tệp tin P2P không cấu trúc.

Áp dụng cơ chế flooding, hãy mô tả cách tìm kiếm một tệp tin khi nút phát yêu cầu có TTL = 3.

Phân tích tình huống: Nếu hệ thống có mật độ nút cao, việc chọn TTL = 3 có thể dẫn đến hệ quả gì về độ bao phủ và chi phí truyền thông?

---

## Trả lời:

### 1. Tổng quan về P2P không cấu trúc và Flooding

**P2P không cấu trúc (Unstructured P2P):**
- Các node kết nối với nhau một cách ngẫu nhiên
- Không có quy tắc về vị trí lưu trữ dữ liệu
- Không có DHT hay cấu trúc định tuyến cố định
- Ví dụ: Gnutella, Kazaa, Freenet

**Flooding (Lan truyền tràn ngập):**
- Cơ chế tìm kiếm đơn giản nhất trong P2P không cấu trúc
- Node gửi query đến TẤT CẢ các neighbors
- Mỗi neighbor tiếp tục gửi đến tất cả neighbors của nó
- Sử dụng **TTL (Time-To-Live)** để giới hạn phạm vi lan truyền

### 2. Mô tả cơ chế Flooding với TTL = 3

#### **Cấu trúc Query Message:**
```
{
  "message_id": "unique-uuid-12345",
  "type": "QUERY",
  "filename": "movie.mp4",
  "ttl": 3,
  "origin_node": "NodeA",
  "hops": 0
}
```

#### **Thuật toán Flooding:**

```
FUNCTION flood_search(query, ttl):
    IF ttl <= 0:
        RETURN  // Dừng lan truyền

    // Kiểm tra xem đã xử lý query này chưa (tránh loop)
    IF query.message_id IN processed_queries:
        RETURN

    // Đánh dấu đã xử lý
    ADD query.message_id TO processed_queries

    // Kiểm tra local: có file không?
    IF file_exists(query.filename):
        SEND query_hit TO query.origin_node

    // Forward đến tất cả neighbors (trừ node gửi đến)
    FOR EACH neighbor IN neighbors:
        new_query = copy(query)
        new_query.ttl = ttl - 1
        new_query.hops = query.hops + 1
        SEND new_query TO neighbor
```

### 3. Ví dụ minh họa với TTL = 3

**Mạng P2P mẫu:**
```
                    [D]
                   / |
                  /  |
        [B]-----[A]--[E]
         |       |    |
         |       |    |
        [C]     [F]--[G]
         |            |
        [H]          [I]
```

**Giả sử:** Node A tìm kiếm file "movie.mp4"

#### **Bước 1: TTL = 3 (tại Node A - Origin)**

Node A gửi query đến tất cả neighbors: B, D, E, F

```
A ──→ B (TTL=2)
A ──→ D (TTL=2)
A ──→ E (TTL=2)
A ──→ F (TTL=2)
```

**Trạng thái:** 4 messages được gửi

#### **Bước 2: TTL = 2 (tại các Node B, D, E, F)**

**Node B** (nhận từ A) → gửi đến neighbors trừ A:
```
B ──→ C (TTL=1)
```

**Node D** (nhận từ A) → gửi đến neighbors trừ A:
```
D ──→ E (TTL=1)  // E cũng là neighbor của D
```

**Node E** (nhận từ A) → gửi đến neighbors trừ A:
```
E ──→ D (TTL=1)  // Nhưng D đã nhận từ A, sẽ bị drop
E ──→ G (TTL=1)
```

**Node F** (nhận từ A) → gửi đến neighbors trừ A:
```
F ──→ G (TTL=1)  // G cũng nhận từ E
```

**Trạng thái:** Thêm ~5 messages (một số bị drop do duplicate)

#### **Bước 3: TTL = 1 (tại các Node C, G)**

**Node C** (nhận từ B):
```
C ──→ H (TTL=0)  // H nhận nhưng không forward tiếp
```

**Node G** (nhận từ E hoặc F, cái nào đến trước):
```
G ──→ I (TTL=0)  // I nhận nhưng không forward tiếp
G ──→ F (TTL=0)  // Có thể bị drop nếu F đã xử lý
```

#### **Bước 4: TTL = 0 (Dừng)**

Các node H, I nhận query với TTL=0:
- Kiểm tra local có file không
- **KHÔNG forward tiếp** (TTL hết)

### 4. Sơ đồ lan truyền

```
Thời gian →

T=0:     [A] Origin
          ↓ TTL=3

T=1:   [B] [D] [E] [F]
          ↓ TTL=2

T=2:   [C]    [G]
          ↓ TTL=1

T=3:   [H]    [I]
          ↓ TTL=0
         STOP
```

### 5. Xử lý Query Hit (Tìm thấy file)

Giả sử Node G có file "movie.mp4":

```
1. Node G nhận query từ E
2. G kiểm tra local → TÌM THẤY "movie.mp4"
3. G gửi QUERY_HIT ngược về theo đường đi:
   G → E → A (hoặc G → F → A)
4. Node A nhận được thông tin:
   - Node G có file
   - Địa chỉ IP/Port của G
5. A kết nối trực tiếp với G để download file
```

**Query Hit Message:**
```
{
  "message_id": "unique-uuid-12345",
  "type": "QUERY_HIT",
  "filename": "movie.mp4",
  "file_size": 1500000000,
  "node_address": "192.168.1.50:6346",
  "hops": 2
}
```

### 6. Phân tích: Mật độ nút cao + TTL = 3

#### **Giả định:**
- Mỗi node có trung bình **k neighbors** (degree)
- TTL = 3
- Mật độ cao: k lớn (ví dụ k = 10)

#### **Ước tính số messages:**

**Công thức (worst case):**
```
Messages ≈ k + k² + k³ = k(k² + k + 1)
```

| TTL | Nodes reached (worst case) | Với k=5 | Với k=10 |
|-----|---------------------------|---------|----------|
| 1 | k | 5 | 10 |
| 2 | k + k² | 30 | 110 |
| 3 | k + k² + k³ | 155 | 1,110 |

#### **Phân tích độ bao phủ (Coverage):**

**Ưu điểm:**
| Khía cạnh | Mô tả |
|-----------|-------|
| ✅ Bao phủ rộng | Với k=10, TTL=3 có thể đạt ~1000 nodes |
| ✅ Tìm nhanh file phổ biến | File có nhiều bản sao sẽ được tìm thấy nhanh |
| ✅ Đơn giản | Không cần cấu trúc phức tạp |

**Nhược điểm:**
| Khía cạnh | Mô tả |
|-----------|-------|
| ❌ Giới hạn phạm vi | Chỉ tìm trong 3 hops, có thể bỏ sót file ở xa |
| ❌ Không đảm bảo | Không chắc chắn tìm thấy dù file tồn tại |

#### **Phân tích chi phí truyền thông (Communication Cost):**

**Vấn đề nghiêm trọng với mật độ cao:**

| Vấn đề | Mô tả | Mức độ |
|--------|-------|--------|
| **Message Explosion** | Số message tăng theo cấp số nhân O(k^TTL) | 🔴 Nghiêm trọng |
| **Bandwidth Consumption** | Mỗi node nhận/gửi hàng trăm messages | 🔴 Nghiêm trọng |
| **Duplicate Messages** | Cùng query đến 1 node qua nhiều đường | 🟡 Trung bình |
| **Processing Overhead** | CPU xử lý nhiều queries | 🟡 Trung bình |

**Ví dụ cụ thể:**
```
Mạng: 10,000 nodes, k=10, TTL=3

Nếu 100 nodes cùng search trong 1 phút:
- Messages/search ≈ 1,000
- Total messages = 100 × 1,000 = 100,000 messages/phút
- Bandwidth: ~10MB/phút (giả sử 100 bytes/message)
```

### 7. Bảng so sánh TTL values

| TTL | Coverage | Messages (k=10) | Trade-off |
|-----|----------|-----------------|-----------|
| 1 | Rất thấp | 10 | Tiết kiệm nhưng kém hiệu quả |
| 2 | Thấp | 110 | Cân bằng cho mạng nhỏ |
| **3** | Trung bình | **1,110** | **Phổ biến, nhưng tốn kém** |
| 4 | Cao | 11,110 | Quá tốn kém |
| 5 | Rất cao | 111,110 | Không khả thi |

### 8. Giải pháp cải thiện

#### **a) Random Walk thay vì Flooding:**
```
Thay vì gửi đến TẤT CẢ neighbors:
→ Chọn NGẪU NHIÊN 1-2 neighbors để forward
→ Giảm messages nhưng vẫn có cơ hội tìm thấy
```

#### **b) Expanding Ring Search:**
```
Bắt đầu với TTL=1
Nếu không tìm thấy → tăng TTL=2
Tiếp tục cho đến khi tìm thấy hoặc TTL max
```

#### **c) Supernode Architecture (Kazaa-style):**
```
- Một số node mạnh làm "supernode"
- Query chỉ flood giữa các supernodes
- Giảm đáng kể số messages
```

#### **d) Bloom Filters:**
```
- Mỗi node lưu Bloom filter của neighbors
- Chỉ forward query đến neighbor có khả năng có file
```

### 9. Kết luận

| Khía cạnh | Đánh giá với TTL=3 và mật độ cao |
|-----------|----------------------------------|
| **Độ bao phủ** | ✅ Tốt - đạt được nhiều nodes trong 3 hops |
| **Chi phí truyền thông** | ❌ Rất cao - O(k³) messages |
| **Scalability** | ❌ Kém - không phù hợp mạng lớn |
| **Khuyến nghị** | Cần kết hợp với các kỹ thuật tối ưu (random walk, supernodes) |

**Tóm lại:** TTL=3 trong mạng mật độ cao tạo ra sự đánh đổi giữa **coverage** và **cost**. Độ bao phủ tốt nhưng chi phí truyền thông rất cao (message explosion), có thể gây quá tải mạng. Cần áp dụng các kỹ thuật tối ưu để cân bằng.
