Câu 8:
Trong một hệ thống phân tán phục vụ đọc nhiều – ghi ít (read heavy, write light) như dịch vụ tập tin, bạn sẽ chọn mô hình đa luồng, đơn luồng, hay máy trạng thái hữu hạn (event driven FSM) cho thành phần xử lý trên máy chủ? Hãy đánh giá dựa trên các tiêu chí: hiệu năng, khả năng mở rộng, độ phức tạp phát triển và khả năng chịu lỗi.

---

## Trả lời:

### 1. Đặc điểm workload "Read Heavy, Write Light"

**Dịch vụ tập tin (File Service) điển hình:**
- **90-99% operations là READ** (đọc file, list directory)
- **1-10% operations là WRITE** (upload, update, delete)
- I/O-bound: phần lớn thời gian chờ disk/network
- Concurrent clients: hàng nghìn đến hàng triệu

```
Workload Profile:
┌────────────────────────────────────────────────┐
│  READ  ████████████████████████████████ 95%   │
│  WRITE ██ 5%                                   │
└────────────────────────────────────────────────┘

Time breakdown per request:
┌────────────────────────────────────────────────┐
│  CPU Processing  ██ 5%                         │
│  I/O Wait       ██████████████████████████ 95% │
└────────────────────────────────────────────────┘
```

### 2. Ba mô hình xử lý

#### **2.1. Mô hình Đơn luồng (Single-Threaded)**

```
┌─────────────────────────────────────────────────┐
│                   SERVER                         │
│  ┌─────────────────────────────────────────┐   │
│  │           Single Thread                  │   │
│  │  while(true):                           │   │
│  │    req = accept()     ─────┐            │   │
│  │    data = read_file()      │ Sequential │   │
│  │    send(data)         ─────┘            │   │
│  └─────────────────────────────────────────┘   │
│              ↓                                  │
│      [Request Queue]                           │
│      R1 → R2 → R3 → R4 → ...                  │
└─────────────────────────────────────────────────┘
```

#### **2.2. Mô hình Đa luồng (Multi-Threaded)**

```
┌─────────────────────────────────────────────────────────┐
│                        SERVER                            │
│  ┌─────────────────────────────────────────────────┐   │
│  │              Thread Pool (N threads)             │   │
│  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐   │   │
│  │  │Thread 1│ │Thread 2│ │Thread 3│ │Thread N│   │   │
│  │  │  R1    │ │  R2    │ │  R3    │ │  R4    │   │   │
│  │  │ (I/O)  │ │ (CPU)  │ │ (I/O)  │ │ (I/O)  │   │   │
│  │  └────────┘ └────────┘ └────────┘ └────────┘   │   │
│  └─────────────────────────────────────────────────┘   │
│                        ↑                                │
│              [Shared Request Queue]                     │
└─────────────────────────────────────────────────────────┘
```

#### **2.3. Mô hình Event-Driven FSM (Finite State Machine)**

```
┌─────────────────────────────────────────────────────────────┐
│                          SERVER                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   Event Loop                         │   │
│  │  while(true):                                        │   │
│  │    events = poll(registered_fds)                    │   │
│  │    for event in events:                             │   │
│  │      handler = state_machine[event.fd]              │   │
│  │      handler.process(event)  // Non-blocking!       │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  State Machine per connection:                              │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐              │
│  │ READING  │ →  │PROCESSING│ →  │ WRITING  │ → Done      │
│  │ request  │    │  data    │    │ response │              │
│  └──────────┘    └──────────┘    └──────────┘              │
└─────────────────────────────────────────────────────────────┘
```

### 3. Đánh giá theo các tiêu chí

#### **3.1. Hiệu năng (Performance)**

| Mô hình | Throughput | Latency | I/O Efficiency |
|---------|------------|---------|----------------|
| **Đơn luồng** | ❌ Thấp | ❌ Cao (blocking) | ❌ Kém |
| **Đa luồng** | ✅ Cao | ✅ Thấp | ✅ Tốt |
| **Event-Driven** | ✅✅ Rất cao | ✅ Thấp | ✅✅ Rất tốt |

**Phân tích chi tiết:**

```
ĐƠN LUỒNG - Performance Analysis:
─────────────────────────────────
Request timeline:
R1: [accept][read file.........][send]
R2:                                    [accept][read][send]
R3:                                                         [...]

Problem: CPU idle 95% thời gian chờ I/O
Throughput: ~50-100 requests/sec (limited by I/O latency)


ĐA LUỒNG - Performance Analysis:
────────────────────────────────
Thread 1: [R1: read file........][R5: read........]
Thread 2: [R2: read file........][R6: read........]
Thread 3: [R3: read file........][R7: read........]
Thread 4: [R4: read file........][R8: read........]

Improvement: Multiple I/O operations in parallel
Throughput: ~500-5000 requests/sec
Overhead: Context switching, memory per thread


EVENT-DRIVEN - Performance Analysis:
────────────────────────────────────
Single thread multiplexing:
Event loop: [R1:accept][R2:accept][R3:accept][R1:data_ready][...]
            └─ Non-blocking, immediate return ─┘

Benefits:
• No context switch overhead
• No thread stack memory (~1MB/thread saved)
• Handles 10,000+ concurrent connections easily

Throughput: ~10,000-100,000 requests/sec
```

**Benchmark ước tính (File Service, 10KB files):**

| Mô hình | Concurrent Connections | Requests/sec | Memory |
|---------|------------------------|--------------|--------|
| Đơn luồng | 1 | 100 | 10 MB |
| Đa luồng (100 threads) | 100 | 2,000 | 100 MB |
| Đa luồng (1000 threads) | 1,000 | 10,000 | 1 GB |
| Event-Driven | 10,000+ | 50,000 | 50 MB |

#### **3.2. Khả năng mở rộng (Scalability)**

| Mô hình | Horizontal | Vertical | Connection Scaling |
|---------|------------|----------|-------------------|
| **Đơn luồng** | ❌ Không | ❌ Không | ❌ O(1) |
| **Đa luồng** | ✅ Có | ✅ Có | ⚠️ O(threads) |
| **Event-Driven** | ✅✅ Tốt | ✅✅ Tốt | ✅✅ O(10K+) |

**Phân tích:**

```
ĐƠN LUỒNG:
• Không scale - 1 request tại 1 thời điểm
• Thêm cores không giúp ích
• Bottleneck: single thread

ĐA LUỒNG:
• Scale với số cores (đến giới hạn)
• Giới hạn:
  - Thread pool size (1000s threads → overhead)
  - Memory: 1MB stack/thread
  - Context switching cost
• C10K problem: 10,000 connections = 10GB RAM

EVENT-DRIVEN:
• Handles C10K, even C100K easily
• Minimal memory per connection (~KB)
• Single thread per core is efficient
• Scale out: multiple event loops on multiple cores
```

```
Connection Scaling Comparison:

Concurrent │                              ╭─ Event-Driven
Connections│                         ╭────╯
           │                    ╭────╯
   10,000+ │               ╭────╯
           │          ╭────╯           ╭─── Multi-Thread
           │     ╭────╯           ╭────╯
    1,000  │╭────╯           ╭────╯
           ││           ╭────╯
      100  ││      ╭────╯
           ││ ╭────╯
       10  │├─╯────────────────────────── Single-Thread
           └┼────────────────────────────────────────→
            1      10     100    1K    10K   Resources
```

#### **3.3. Độ phức tạp phát triển (Development Complexity)**

| Mô hình | Code Complexity | Debugging | Reasoning |
|---------|-----------------|-----------|-----------|
| **Đơn luồng** | ⭐⭐⭐⭐⭐ Rất đơn giản | ⭐⭐⭐⭐⭐ Dễ | Sequential |
| **Đa luồng** | ⭐⭐ Phức tạp | ⭐⭐ Khó | Concurrent |
| **Event-Driven** | ⭐⭐⭐ Trung bình | ⭐⭐⭐ Trung bình | State machine |

**So sánh code:**

```python
# ĐƠN LUỒNG - Simple, sequential
def handle_request(socket):
    request = socket.recv()        # Blocking
    data = read_file(request.path) # Blocking
    socket.send(data)              # Blocking
    socket.close()

while True:
    client = server.accept()       # Blocking
    handle_request(client)
```

```python
# ĐA LUỒNG - Concurrent, needs synchronization
from threading import Thread, Lock

file_cache = {}
cache_lock = Lock()

def handle_request(socket):
    request = socket.recv()

    with cache_lock:               # Critical section
        if request.path in file_cache:
            data = file_cache[request.path]
        else:
            data = read_file(request.path)
            file_cache[request.path] = data

    socket.send(data)
    socket.close()

while True:
    client = server.accept()
    Thread(target=handle_request, args=(client,)).start()

# Challenges: Race conditions, deadlocks, debugging
```

```python
# EVENT-DRIVEN - State machine, callback-based
import select

class ConnectionState:
    READING_REQUEST = 1
    READING_FILE = 2
    SENDING_RESPONSE = 3

connections = {}  # fd -> state

def event_loop():
    while True:
        readable, writable, _ = select.select(read_fds, write_fds, [])

        for fd in readable:
            state = connections[fd]
            if state.phase == READING_REQUEST:
                data = fd.recv(1024)  # Non-blocking
                if complete_request(data):
                    state.phase = READING_FILE
                    start_async_read(state.path)
            elif state.phase == READING_FILE:
                # File data ready
                state.response = state.file_data
                state.phase = SENDING_RESPONSE

        for fd in writable:
            if connections[fd].phase == SENDING_RESPONSE:
                sent = fd.send(state.response)  # Non-blocking
                if all_sent:
                    cleanup(fd)

# Challenges: Callback hell, state management
```

#### **3.4. Khả năng chịu lỗi (Fault Tolerance)**

| Mô hình | Error Isolation | Recovery | Partial Failure |
|---------|-----------------|----------|-----------------|
| **Đơn luồng** | ⭐⭐⭐⭐⭐ Tốt nhất | ✅ Đơn giản | ❌ Crash all |
| **Đa luồng** | ⭐⭐ Kém | ⚠️ Phức tạp | ⚠️ Có thể crash process |
| **Event-Driven** | ⭐⭐⭐⭐ Tốt | ✅ Tốt | ✅ Isolated per connection |

**Phân tích:**

```
ĐƠN LUỒNG:
• Error trong 1 request → chỉ request đó fail
• Dễ recover: try-catch, restart loop
• Nhược điểm: 1 request treo → cả server treo

ĐA LUỒNG:
• Error trong 1 thread có thể:
  - Corrupt shared state
  - Deadlock các threads khác
  - Crash toàn bộ process (uncaught exception)
• Khó debug race conditions
• Cần careful exception handling

EVENT-DRIVEN:
• Error trong 1 connection handler:
  - Cleanup connection đó
  - Các connections khác không bị ảnh hưởng
• Single point of failure: event loop crash → all down
• Solution: Multiple event loops, process supervision
```

### 4. Bảng tổng hợp đánh giá

| Tiêu chí | Đơn luồng | Đa luồng | Event-Driven |
|----------|:---------:|:--------:|:------------:|
| **Hiệu năng** | ⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Khả năng mở rộng** | ⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Độ phức tạp phát triển** | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ |
| **Khả năng chịu lỗi** | ⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐ |
| **TỔNG** | 10/20 | 11/20 | **17/20** |

### 5. Khuyến nghị cho File Service

```
┌─────────────────────────────────────────────────────────────────┐
│  KHUYẾN NGHỊ: EVENT-DRIVEN FSM                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Lý do chọn Event-Driven cho Read-Heavy File Service:          │
│                                                                  │
│  1. ✅ I/O-bound workload phù hợp hoàn hảo                      │
│     - 95% thời gian chờ I/O → event-driven tận dụng tốt        │
│                                                                  │
│  2. ✅ Handles thousands of concurrent readers                  │
│     - File service cần serve nhiều clients đồng thời            │
│     - Event-driven: C10K+ với minimal resources                 │
│                                                                  │
│  3. ✅ Read operations are independent                          │
│     - Không cần synchronization phức tạp như write              │
│     - State machine đơn giản: ACCEPT → READ → SEND              │
│                                                                  │
│  4. ✅ Memory efficient                                          │
│     - Quan trọng khi có nhiều connections                       │
│                                                                  │
│  5. ✅ Fault isolation per connection                           │
│     - 1 connection lỗi không ảnh hưởng các connections khác    │
│                                                                  │
├─────────────────────────────────────────────────────────────────┤
│  Hybrid approach (Production):                                   │
│  • Event loop + Thread pool for file I/O                        │
│  • Multiple event loops (1 per core) + work stealing            │
│  • Examples: Nginx, Redis, Node.js                              │
└─────────────────────────────────────────────────────────────────┘
```

### 6. Kiến trúc đề xuất

```
┌──────────────────────────────────────────────────────────────────┐
│                    FILE SERVICE ARCHITECTURE                      │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                     Load Balancer                           │ │
│  └────────────────────────────────────────────────────────────┘ │
│                    ↓           ↓           ↓                     │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │              Event Loop Workers (N = num_cores)           │   │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────┐            │   │
│  │  │Event Loop 1│ │Event Loop 2│ │Event Loop N│            │   │
│  │  │ (Core 1)   │ │ (Core 2)   │ │ (Core N)   │            │   │
│  │  └─────┬──────┘ └─────┬──────┘ └─────┬──────┘            │   │
│  │        └──────────────┼──────────────┘                    │   │
│  │                       ↓                                    │   │
│  │              ┌─────────────────┐                          │   │
│  │              │  Async I/O Pool │ ← libuv, io_uring        │   │
│  │              └────────┬────────┘                          │   │
│  └───────────────────────┼──────────────────────────────────┘   │
│                          ↓                                       │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                    File System / Cache                      │ │
│  └────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

### 7. Kết luận

Với đặc điểm **read-heavy, write-light** và **I/O-bound** của dịch vụ tập tin:

| Lựa chọn | Đánh giá |
|----------|----------|
| Đơn luồng | ❌ Không phù hợp - quá chậm |
| Đa luồng | ⚠️ Có thể dùng - nhưng tốn resources |
| **Event-Driven FSM** | ✅ **Phù hợp nhất** - hiệu năng cao, scale tốt |

**Ví dụ thực tế:** Nginx (web server, file serving) sử dụng event-driven architecture và có thể xử lý hàng chục nghìn connections đồng thời với minimal resources.
