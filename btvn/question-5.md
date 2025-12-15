Câu 5:

Trong một mạng P2P không cấu trúc, một nút cần tìm kiếm dữ liệu hiếm (rare data).

Hãy áp dụng phương pháp random walk để mô tả cách tìm kiếm dữ liệu này.

Đề xuất và giải thích cách cải tiến (ví dụ: khởi động nhiều random walks đồng thời) để giảm thời gian tìm thấy dữ liệu, và phân tích sự đánh đổi giữa thời gian tìm kiếm và lưu lượng mạng.

---

## Trả lời:

### 1. Tổng quan về Random Walk trong P2P

**Random Walk** là phương pháp tìm kiếm trong mạng P2P không cấu trúc, trong đó query message di chuyển ngẫu nhiên từ node này sang node khác cho đến khi tìm thấy dữ liệu hoặc đạt giới hạn.

**So sánh với Flooding:**
| Tiêu chí | Flooding | Random Walk |
|----------|----------|-------------|
| Số messages | O(k^TTL) - rất cao | O(TTL) - tuyến tính |
| Coverage | Rộng, đồng thời | Hẹp, tuần tự |
| Phù hợp | Dữ liệu phổ biến | Dữ liệu hiếm (với cải tiến) |

### 2. Cơ chế Random Walk cơ bản

#### **Thuật toán:**

```python
def random_walk_search(query, ttl, max_ttl):
    """
    query: thông tin tìm kiếm (filename, keywords)
    ttl: số bước còn lại
    max_ttl: giới hạn tối đa
    """

    # Điều kiện dừng
    if ttl <= 0:
        return NOT_FOUND

    # Kiểm tra local
    if has_data(query.filename):
        return FOUND(current_node)

    # Chọn NGẪU NHIÊN 1 neighbor
    next_node = random.choice(neighbors)

    # Forward query
    query.ttl = ttl - 1
    query.path.append(current_node)

    return forward(query, next_node)
```

#### **Ví dụ minh họa:**

```
Mạng P2P:
    [A]---[B]---[C]---[D]
     |     |     |     |
    [E]---[F]---[G]---[H]
     |     |     |     |
    [I]---[J]---[K]---[L]  ← L có dữ liệu hiếm

Node A tìm kiếm dữ liệu, TTL = 10
```

**Random Walk có thể đi theo đường:**
```
A → B → F → G → C → D → H → L (FOUND!)
     ↓
   7 hops, 7 messages
```

**So với Flooding TTL=3:**
```
Flooding: ~100+ messages nhưng có thể không đến được L
Random Walk: 7 messages, đến được L
```

### 3. Vấn đề với dữ liệu hiếm (Rare Data)

**Thách thức:**
- Dữ liệu hiếm chỉ có ở 1-2 nodes trong mạng lớn
- Random walk đơn lẻ có thể mất rất lâu để tìm thấy
- Xác suất tìm thấy trong k bước: P = 1 - (1 - r)^k
  - Với r = tỷ lệ nodes có dữ liệu (rất nhỏ với rare data)

**Ví dụ:**
```
Mạng 10,000 nodes, chỉ 10 nodes có dữ liệu
r = 10/10,000 = 0.001 (0.1%)

Xác suất tìm thấy trong 100 bước:
P = 1 - (1-0.001)^100 ≈ 9.5%

Xác suất tìm thấy trong 1000 bước:
P = 1 - (1-0.001)^1000 ≈ 63%
```

→ **Cần cải tiến để tìm nhanh hơn!**

### 4. Các phương pháp cải tiến

#### **4.1. Multiple Random Walks (k-Random Walks)**

**Ý tưởng:** Khởi động k random walks đồng thời từ node nguồn.

```python
def k_random_walks(query, k, ttl):
    """
    k: số random walks đồng thời
    """
    results = []

    # Chọn k neighbors khác nhau (hoặc có thể trùng)
    selected_neighbors = random.sample(neighbors, min(k, len(neighbors)))

    # Khởi động k walks song song
    for i, neighbor in enumerate(selected_neighbors):
        walk_query = copy(query)
        walk_query.walk_id = i
        walk_query.ttl = ttl

        # Gửi đồng thời
        async_send(walk_query, neighbor)

    # Chờ kết quả đầu tiên (hoặc timeout)
    return wait_for_first_result(timeout=30s)
```

**Sơ đồ:**
```
         ┌──→ Walk 1: A→B→F→J→...
         │
Node A ──┼──→ Walk 2: A→E→I→J→K→L (FOUND!)
         │
         └──→ Walk 3: A→B→C→G→...
```

**Phân tích:**
- **Thời gian:** Giảm đáng kể (chia cho k nếu độc lập)
- **Messages:** Tăng k lần so với single walk
- **Xác suất tìm thấy:** P_k = 1 - (1-r)^(k×steps)

#### **4.2. Biased Random Walk**

**Ý tưởng:** Không chọn neighbor hoàn toàn ngẫu nhiên, mà ưu tiên nodes có khả năng cao hơn.

```python
def biased_random_walk(query, ttl):
    if ttl <= 0 or has_data(query):
        return result

    # Tính điểm cho mỗi neighbor
    scores = {}
    for neighbor in neighbors:
        scores[neighbor] = calculate_score(neighbor, query)

    # Chọn neighbor với xác suất tỷ lệ với score
    next_node = weighted_random_choice(neighbors, scores)

    return forward(query, next_node)

def calculate_score(neighbor, query):
    """Các tiêu chí đánh giá"""
    score = 1.0

    # Ưu tiên node có nhiều connections (high degree)
    score *= neighbor.degree / avg_degree

    # Ưu tiên node chưa được visit gần đây
    if neighbor not in recent_visits:
        score *= 2.0

    # Ưu tiên dựa trên metadata (nếu có)
    if query.category in neighbor.content_categories:
        score *= 3.0

    return score
```

#### **4.3. Random Walk với Checkpointing**

**Ý tưởng:** Lưu lại các nodes đã visit, tránh lặp lại.

```python
def random_walk_with_history(query, ttl, visited_set):
    if ttl <= 0:
        return NOT_FOUND

    if has_data(query):
        return FOUND

    # Thêm current node vào visited
    visited_set.add(current_node)

    # Chọn neighbor CHƯA VISIT
    unvisited = [n for n in neighbors if n not in visited_set]

    if not unvisited:
        # Backtrack hoặc random restart
        return random_restart(query, ttl)

    next_node = random.choice(unvisited)
    return forward(query, next_node, visited_set)
```

#### **4.4. Adaptive Random Walk**

**Ý tưởng:** Điều chỉnh số walks dựa trên độ hiếm của dữ liệu.

```python
def adaptive_random_walks(query):
    # Bắt đầu với ít walks
    k = 2
    ttl = 50

    while not found and k <= MAX_WALKS:
        # Thử với k walks
        result = k_random_walks(query, k, ttl)

        if result == FOUND:
            return result

        # Tăng số walks
        k = k * 2

        # Tùy chọn: tăng TTL
        ttl = min(ttl + 20, MAX_TTL)

    return NOT_FOUND
```

### 5. Phân tích đánh đổi: Thời gian vs Lưu lượng mạng

#### **Bảng so sánh các phương pháp:**

| Phương pháp | Messages | Thời gian | Trade-off |
|-------------|----------|-----------|-----------|
| Single Walk | TTL | Cao | Tiết kiệm bandwidth, chậm |
| k-Walks (k=4) | 4×TTL | Giảm ~4x | Cân bằng tốt |
| k-Walks (k=16) | 16×TTL | Giảm ~16x | Tốn bandwidth |
| Flooding TTL=3 | O(k³) | Thấp nhất | Rất tốn, không scale |

#### **Đồ thị Trade-off:**

```
Thời gian tìm kiếm
     ↑
     │
Cao  │  ●─── Single Random Walk
     │    ╲
     │      ╲
     │        ●─── 2-Random Walks
     │          ╲
     │            ●─── 4-Random Walks
     │              ╲
     │                ●─── 8-Random Walks
Thấp │                  ╲
     │                    ●─── 16-Random Walks
     └──────────────────────────────→
     Thấp                        Cao
              Lưu lượng mạng (Messages)
```

#### **Phân tích chi tiết:**

| Số walks (k) | Messages (TTL=100) | Thời gian trung bình | Xác suất (r=0.1%) |
|--------------|--------------------|-----------------------|-------------------|
| 1 | 100 | 1000 steps | 9.5% |
| 2 | 200 | 500 steps | 18.1% |
| 4 | 400 | 250 steps | 33.0% |
| 8 | 800 | 125 steps | 55.1% |
| 16 | 1,600 | 62 steps | 79.8% |
| 32 | 3,200 | 31 steps | 95.9% |

### 6. Khuyến nghị cho Rare Data

```
┌─────────────────────────────────────────────────────────┐
│  KHUYẾN NGHỊ: Kết hợp nhiều kỹ thuật                   │
├─────────────────────────────────────────────────────────┤
│  1. Sử dụng k-Random Walks với k = 4-8                 │
│  2. Áp dụng Biased Walk (ưu tiên high-degree nodes)    │
│  3. Tracking visited nodes để tránh lặp                │
│  4. Adaptive: tăng k nếu chưa tìm thấy                 │
│  5. TTL hợp lý: 50-100 cho mạng lớn                    │
└─────────────────────────────────────────────────────────┘
```

### 7. Kết luận

| Tiêu chí | Đánh giá |
|----------|----------|
| **Random Walk cơ bản** | Tiết kiệm bandwidth nhưng chậm với rare data |
| **k-Random Walks** | Giải pháp hiệu quả, cân bằng tốt |
| **Trade-off** | Tăng k → giảm thời gian, tăng messages (tuyến tính) |
| **So với Flooding** | Tốt hơn nhiều về scalability (O(k×TTL) vs O(degree^TTL)) |

**Công thức tối ưu:**
```
k_optimal ≈ √(N/n)
Trong đó:
- N = tổng số nodes
- n = số nodes có dữ liệu
```
