Câu 3:

Trong vòng Chord với m = 5 và các nút hiện có {1, 4, 9, 11, 14, 18, 20, 21, 28}, hãy:

Xác định succ(7), succ(22) và succ(30).

Giả sử node 9 thực hiện tra cứu key = 3, hãy mô tả chi tiết các bước chuyển tiếp yêu cầu qua các shortcut (theo hình 2.19) cho đến khi tìm được node chịu trách nhiệm.

---

## Trả lời:

### 1. Tổng quan về Chord DHT

**Chord** là một giao thức Distributed Hash Table (DHT) có cấu trúc, sử dụng consistent hashing để phân phối keys cho các nodes trong một vòng tròn logic.

**Các thông số:**
- **m = 5** → Không gian định danh: 2^5 = **32** (từ 0 đến 31)
- **Nodes hiện có:** {1, 4, 9, 11, 14, 18, 20, 21, 28}
- **Số nodes:** 9

### 2. Biểu diễn vòng Chord (m=5)

```
                           0
                      31       1 ←[NODE]
                   30            2
                29                 3
              28 ←[NODE]             4 ←[NODE]
             27                        5
            26                          6
           25                            7
          24                              8
          23                              9 ←[NODE]
           22                            10
            21 ←[NODE]                  11 ←[NODE]
             20 ←[NODE]                12
              19                     13
                18 ←[NODE]         14 ←[NODE]
                   17            15
                      16      16

Nodes: {1, 4, 9, 11, 14, 18, 20, 21, 28}
```

### 3. Xác định Successor

**Định nghĩa:** `succ(k)` = node đầu tiên có ID ≥ k khi đi theo chiều kim đồng hồ trên vòng Chord.

#### **succ(7) = ?**

Đi từ vị trí 7 theo chiều kim đồng hồ, tìm node đầu tiên:
- Vị trí 7 → không có node
- Vị trí 8 → không có node
- Vị trí 9 → **có NODE 9** ✓

**→ succ(7) = 9**

#### **succ(22) = ?**

Đi từ vị trí 22 theo chiều kim đồng hồ:
- Vị trí 22 → không có node
- Vị trí 23 → không có node
- ...
- Vị trí 28 → **có NODE 28** ✓

**→ succ(22) = 28**

#### **succ(30) = ?**

Đi từ vị trí 30 theo chiều kim đồng hồ:
- Vị trí 30 → không có node
- Vị trí 31 → không có node
- Vị trí 0 → không có node (wrap around)
- Vị trí 1 → **có NODE 1** ✓

**→ succ(30) = 1** (wrap around qua 0)

### 4. Bảng tóm tắt Successor

| Key (k) | Tìm kiếm | Kết quả |
|---------|----------|---------|
| succ(7) | 7 → 8 → **9** | **9** |
| succ(22) | 22 → 23 → ... → **28** | **28** |
| succ(30) | 30 → 31 → 0 → **1** | **1** |

### 5. Finger Table của Node 9

Để tra cứu key = 3 từ node 9, cần xây dựng **Finger Table** của node 9.

**Công thức:** `finger[i] = succ((n + 2^(i-1)) mod 2^m)` với i = 1, 2, ..., m

Với n = 9, m = 5:

| i | start = (9 + 2^(i-1)) mod 32 | finger[i] = succ(start) |
|---|------------------------------|-------------------------|
| 1 | (9 + 1) mod 32 = 10 | succ(10) = **11** |
| 2 | (9 + 2) mod 32 = 11 | succ(11) = **11** |
| 3 | (9 + 4) mod 32 = 13 | succ(13) = **14** |
| 4 | (9 + 8) mod 32 = 17 | succ(17) = **18** |
| 5 | (9 + 16) mod 32 = 25 | succ(25) = **28** |

**Finger Table của Node 9:**
```
┌───────┬───────┬────────────┐
│   i   │ start │ finger[i]  │
├───────┼───────┼────────────┤
│   1   │  10   │     11     │
│   2   │  11   │     11     │
│   3   │  13   │     14     │
│   4   │  17   │     18     │
│   5   │  25   │     28     │
└───────┴───────┴────────────┘
```

### 6. Tra cứu Key = 3 từ Node 9

**Mục tiêu:** Tìm node chịu trách nhiệm cho key = 3 (tức là succ(3) = **4**)

#### **Thuật toán Chord Lookup:**
1. Nếu key nằm trong khoảng (n, successor], trả về successor
2. Ngược lại, chuyển tiếp đến node gần key nhất trong finger table

#### **Bước 1: Tại Node 9**

- Key = 3
- Node 9 kiểm tra: key = 3 có nằm trong (9, succ(9)] = (9, 11] không?
- Vì không gian là vòng tròn, khoảng (9, 11] không chứa 3
- **Cần chuyển tiếp request**

Tìm finger lớn nhất < key = 3:
- Duyệt finger table từ i=5 xuống i=1:
  - finger[5] = 28: 28 > 3? Trong không gian vòng, cần xét khoảng (9, 3)
  - Khoảng (9, 3) theo chiều kim đồng hồ: 10, 11, ..., 31, 0, 1, 2
  - finger[5] = 28 ∈ (9, 3)? **Có** → Chuyển đến **node 28**

#### **Bước 2: Tại Node 28**

Cần xây dựng Finger Table của Node 28:

| i | start = (28 + 2^(i-1)) mod 32 | finger[i] |
|---|-------------------------------|-----------|
| 1 | 29 | succ(29) = **1** |
| 2 | 30 | succ(30) = **1** |
| 3 | 0 | succ(0) = **1** |
| 4 | 4 | succ(4) = **4** |
| 5 | 12 | succ(12) = **14** |

- Node 28 kiểm tra: key = 3 ∈ (28, succ(28)] = (28, 1]?
- Khoảng (28, 1] theo chiều kim đồng hồ: 29, 30, 31, 0, 1
- 3 ∉ (28, 1] → **Cần chuyển tiếp**

Tìm finger gần key nhất trong khoảng (28, 3):
- finger[5] = 14: 14 ∈ (28, 3)? Không (14 không nằm trong 29,30,31,0,1,2)
- finger[4] = 4: 4 ∈ (28, 3)? Không
- finger[3] = 1: 1 ∈ (28, 3)? **Có** (1 nằm trong 29,...,0,1,2)
- Chuyển đến **node 1**

#### **Bước 3: Tại Node 1**

Finger Table của Node 1:

| i | start = (1 + 2^(i-1)) mod 32 | finger[i] |
|---|------------------------------|-----------|
| 1 | 2 | succ(2) = **4** |
| 2 | 3 | succ(3) = **4** |
| 3 | 5 | succ(5) = **9** |
| 4 | 9 | succ(9) = **9** |
| 5 | 17 | succ(17) = **18** |

- Node 1 kiểm tra: key = 3 ∈ (1, succ(1)] = (1, 4]?
- 3 ∈ (1, 4]? **CÓ!** ✓
- **Trả về successor = 4**

### 7. Tóm tắt quá trình tra cứu

```
Node 9 ──[finger[5]=28]──→ Node 28 ──[finger[3]=1]──→ Node 1 ──→ Found: Node 4
```

| Bước | Tại Node | Hành động | Chuyển đến |
|------|----------|-----------|------------|
| 1 | 9 | key=3 ∉ (9,11], dùng finger[5]=28 | Node 28 |
| 2 | 28 | key=3 ∉ (28,1], dùng finger[3]=1 | Node 1 |
| 3 | 1 | key=3 ∈ (1,4], trả về succ=4 | **Node 4** |

### 8. Kết luận

- **succ(7) = 9**
- **succ(22) = 28**
- **succ(30) = 1**
- **Key 3 được lưu tại Node 4**
- Số bước lookup: **3 hops** (9 → 28 → 1 → 4)
- Độ phức tạp: O(log N) với N = 9 nodes, log₂(9) ≈ 3.17 ✓