Câu 7:
So sánh hai cách cài đặt luồng: (a) hoàn toàn ở mức người dùng (user level threads) và (b) kết hợp luồng ở mức người dùng với lightweight process (LWP) ở mức kernel. Phân tích ưu, nhược điểm của từng cách về chi phí chuyển ngữ cảnh, khả năng phong tỏa và độ phức tạp triển khai.

---

## Trả lời:

### 1. Tổng quan các mô hình Thread

```
┌─────────────────────────────────────────────────────────────────────┐
│                    CÁC MÔ HÌNH THREAD                                │
├─────────────────┬─────────────────────┬─────────────────────────────┤
│  User-Level     │   Kernel-Level      │   Hybrid (M:N)              │
│  Threads (ULT)  │   Threads (KLT)     │   User + LWP               │
│  Many-to-One    │   One-to-One        │   Many-to-Many              │
└─────────────────┴─────────────────────┴─────────────────────────────┘
```

### 2. Mô hình (a): User-Level Threads (ULT) - Many-to-One

#### **Kiến trúc:**

```
┌─────────────────────────────────────────┐
│              USER SPACE                  │
│  ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐       │
│  │ T1  │ │ T2  │ │ T3  │ │ T4  │       │
│  └──┬──┘ └──┬──┘ └──┬──┘ └──┬──┘       │
│     └───────┴───────┴───────┘           │
│              ↓                           │
│     ┌─────────────────┐                 │
│     │  Thread Library │ ← Quản lý threads│
│     │  (Scheduler)    │   trong user space│
│     └────────┬────────┘                 │
└──────────────┼──────────────────────────┘
               ↓
┌──────────────┼──────────────────────────┐
│              ↓         KERNEL SPACE      │
│     ┌─────────────────┐                 │
│     │  Kernel Thread  │ ← Kernel chỉ    │
│     │  (1 per process)│   thấy 1 thread │
│     └─────────────────┘                 │
└─────────────────────────────────────────┘
```

**Đặc điểm:**
- Thread được quản lý hoàn toàn bởi **user-space library** (pthreads, green threads)
- Kernel **không biết** về sự tồn tại của các threads
- Tất cả user threads map vào **1 kernel thread**
- Ví dụ: GNU Portable Threads, early Java Green Threads

#### **Cơ chế hoạt động:**

```python
# Thread Library (User Space)
class UserThreadScheduler:
    def __init__(self):
        self.threads = []
        self.current = None

    def create_thread(self, func):
        thread = UserThread(func)
        self.threads.append(thread)

    def yield_thread(self):
        # Lưu context của current thread
        save_context(self.current)

        # Chọn thread tiếp theo (round-robin)
        self.current = self.pick_next()

        # Restore context và chạy
        restore_context(self.current)

    def schedule(self):
        # Cooperative scheduling
        # Thread phải tự gọi yield()
        while self.threads:
            self.current.run_until_yield()
```

### 3. Mô hình (b): Hybrid Model - User Threads + LWP (Many-to-Many)

#### **Kiến trúc:**

```
┌─────────────────────────────────────────────────────────────┐
│                      USER SPACE                              │
│  ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐          │
│  │UT 1 │ │UT 2 │ │UT 3 │ │UT 4 │ │UT 5 │ │UT 6 │          │
│  └──┬──┘ └──┬──┘ └──┬──┘ └──┬──┘ └──┬──┘ └──┬──┘          │
│     │       │       │       │       │       │               │
│     └───┬───┴───────┼───────┴───┬───┘       │               │
│         ↓           ↓           ↓           ↓               │
│     ┌───────────────────────────────────────────┐          │
│     │         Thread Library + Scheduler         │          │
│     └───────────┬───────────────┬───────────────┘          │
│                 ↓               ↓                           │
│     ┌───────────────┐   ┌───────────────┐                  │
│     │     LWP 1     │   │     LWP 2     │   ← User-visible │
│     │ (bound to K1) │   │ (bound to K2) │     kernel threads│
│     └───────┬───────┘   └───────┬───────┘                  │
└─────────────┼───────────────────┼───────────────────────────┘
              ↓                   ↓
┌─────────────┼───────────────────┼───────────────────────────┐
│             ↓   KERNEL SPACE    ↓                           │
│     ┌───────────────┐   ┌───────────────┐                  │
│     │ Kernel Thread │   │ Kernel Thread │                  │
│     │      K1       │   │      K2       │                  │
│     └───────────────┘   └───────────────┘                  │
│                    ↓         ↓                              │
│              ┌─────────────────────┐                        │
│              │   Kernel Scheduler  │                        │
│              └─────────────────────┘                        │
└─────────────────────────────────────────────────────────────┘
```

**Đặc điểm:**
- **M user threads** được map vào **N kernel threads (LWPs)** với M ≥ N
- LWP (Lightweight Process) = kernel thread có thể được lập lịch bởi kernel
- Thread library quản lý việc gán user threads vào LWPs
- Ví dụ: Solaris Threads, IRIX, HP-UX

#### **Cơ chế Scheduler Activation:**

```
┌────────────────────────────────────────────────────────────┐
│  SCHEDULER ACTIVATIONS                                      │
├────────────────────────────────────────────────────────────┤
│  1. Kernel thông báo cho user-space khi:                   │
│     - LWP bị block (blocked activation)                    │
│     - LWP được unblock (unblocked activation)              │
│     - Có thêm processor available                          │
│                                                            │
│  2. User-space scheduler quyết định:                       │
│     - Gán user thread nào cho LWP nào                      │
│     - Khi nào preempt user thread                          │
└────────────────────────────────────────────────────────────┘
```

### 4. So sánh chi tiết

#### **4.1. Chi phí chuyển ngữ cảnh (Context Switch Cost)**

| Tiêu chí | User-Level Threads | Hybrid (ULT + LWP) |
|----------|-------------------|---------------------|
| **Intra-process switch** | **Rất nhanh** (~100 cycles) | Nhanh (~100-500 cycles) |
| **Cần kernel call?** | Không | Không (giữa ULTs trên cùng LWP) |
| **Thao tác** | Save/restore: PC, SP, registers | Tương tự + có thể đổi LWP |
| **TLB/Cache flush** | Không | Không |
| **Cross-LWP switch** | N/A | Cần kernel intervention |

**Chi tiết:**

```
USER-LEVEL THREAD SWITCH (ULT):
┌────────────────────────────────────────┐
│ 1. Save registers của T1 (user space) │ ~20 cycles
│ 2. Update thread table                 │ ~10 cycles
│ 3. Select next thread T2               │ ~20 cycles
│ 4. Restore registers của T2            │ ~20 cycles
│ ─────────────────────────────────────  │
│ TOTAL: ~70-100 cycles                  │
│ NO KERNEL INVOLVEMENT                  │
└────────────────────────────────────────┘

HYBRID MODEL SWITCH:
┌────────────────────────────────────────────────┐
│ Case 1: Switch giữa ULTs trên cùng LWP        │
│ → Giống ULT, ~100 cycles                       │
│                                                │
│ Case 2: Switch khi ULT block (syscall)        │
│ 1. ULT1 calls blocking syscall                │
│ 2. Kernel blocks LWP1                          │
│ 3. Scheduler activation notifies user library │
│ 4. Library schedules ULT2 on LWP2             │
│ → ~1000-5000 cycles (kernel involved)         │
└────────────────────────────────────────────────┘
```

#### **4.2. Khả năng phong tỏa (Blocking Behavior)**

| Tiêu chí | User-Level Threads | Hybrid (ULT + LWP) |
|----------|-------------------|---------------------|
| **Khi 1 thread gọi blocking I/O** | ❌ **Toàn bộ process bị block** | ✅ Chỉ LWP đó bị block |
| **Các threads khác** | Không chạy được | Chạy trên LWPs khác |
| **Page fault** | Block toàn bộ | Chỉ block LWP liên quan |
| **Giải pháp cho ULT** | Non-blocking I/O, async I/O | Không cần |

**Minh họa vấn đề blocking:**

```
USER-LEVEL THREADS - Blocking Problem:
─────────────────────────────────────
Time →
T1: ████████░░░░░░░░░░░░░░░░░░░░████████
T2: ────────░░░░░░░░░░░░░░░░░░░░────────  BLOCKED!
T3: ────────░░░░░░░░░░░░░░░░░░░░────────  BLOCKED!
         ↑                    ↑
    T1 calls               I/O done
    read()

Kernel chỉ thấy 1 thread → block cả process!


HYBRID MODEL - No Blocking Problem:
───────────────────────────────────
T1 on LWP1: ████████░░░░░░░░░░░░████████
T2 on LWP2: ────────████████████────────  RUNNING!
T3 on LWP2: ────────────────────████████  RUNNING!
                ↑           ↑
           LWP1 blocked   LWP2 continues
```

#### **4.3. Độ phức tạp triển khai (Implementation Complexity)**

| Tiêu chí | User-Level Threads | Hybrid (ULT + LWP) |
|----------|-------------------|---------------------|
| **Độ phức tạp** | ✅ **Thấp** | ❌ **Cao** |
| **Cần sửa kernel?** | Không | Có (scheduler activations) |
| **Thread library** | Đơn giản | Phức tạp |
| **Debugging** | Dễ hơn | Khó hơn |
| **Portability** | Cao (pure user space) | Thấp (kernel-dependent) |

**Chi tiết implementation:**

```
USER-LEVEL THREADS:
┌────────────────────────────────────────────────┐
│ Components needed:                              │
│ • Thread control blocks (TCB) in user space   │
│ • Simple round-robin or priority scheduler    │
│ • setjmp/longjmp hoặc assembly cho switching  │
│ • Mutex/Condition variables (user space)      │
│                                                │
│ LOC estimate: ~2000-5000 lines                │
└────────────────────────────────────────────────┘

HYBRID MODEL:
┌────────────────────────────────────────────────┐
│ Components needed:                              │
│ • Everything from ULT                          │
│ • Kernel support for LWP creation              │
│ • Scheduler activations in kernel              │
│ • Upcall mechanism (kernel → user)             │
│ • LWP pool management                          │
│ • Complex synchronization                      │
│                                                │
│ LOC estimate: ~10000-50000 lines              │
│ (kernel + user space)                          │
└────────────────────────────────────────────────┘
```

### 5. Bảng tổng hợp so sánh

| Tiêu chí | User-Level Threads | Hybrid (ULT + LWP) |
|----------|:------------------:|:-------------------:|
| **Context Switch Cost** | ⭐⭐⭐⭐⭐ Rất nhanh | ⭐⭐⭐⭐ Nhanh |
| **Blocking Behavior** | ⭐ Tệ (block all) | ⭐⭐⭐⭐⭐ Tốt |
| **True Parallelism** | ❌ Không | ✅ Có |
| **Implementation** | ⭐⭐⭐⭐⭐ Đơn giản | ⭐⭐ Phức tạp |
| **Portability** | ⭐⭐⭐⭐⭐ Cao | ⭐⭐ Thấp |
| **Kernel Dependency** | Không | Có |
| **Scalability** | Thấp (1 core) | Cao (multi-core) |
| **Use Cases** | Cooperative tasks, coroutines | General purpose |

### 6. Ví dụ thực tế

| Mô hình | Implementations |
|---------|-----------------|
| **User-Level** | GNU Pth, Early Java Green Threads, Stackless Python, Goroutines (Go - hybrid) |
| **Hybrid M:N** | Solaris Threads, FreeBSD KSE, Windows Fibers + Threads |
| **1:1 (so sánh)** | Linux NPTL, Windows Threads, Modern Java |

### 7. Kết luận và Khuyến nghị

```
┌─────────────────────────────────────────────────────────────────┐
│                        KHUYẾN NGHỊ                               │
├─────────────────────────────────────────────────────────────────┤
│  Chọn USER-LEVEL THREADS khi:                                   │
│  • Ứng dụng CPU-bound, ít I/O blocking                         │
│  • Cần context switch cực nhanh                                 │
│  • Portability quan trọng                                       │
│  • Cooperative multitasking đủ dùng                             │
│                                                                  │
│  Chọn HYBRID (ULT + LWP) khi:                                   │
│  • Ứng dụng I/O-bound với nhiều blocking calls                 │
│  • Cần true parallelism trên multi-core                        │
│  • Có thể chấp nhận kernel dependency                          │
│  • Performance và scalability quan trọng                        │
├─────────────────────────────────────────────────────────────────┤
│  Xu hướng hiện đại:                                              │
│  • Hầu hết OS dùng 1:1 (NPTL) vì đơn giản và hiệu quả          │
│  • M:N phức tạp nhưng vẫn dùng ở Go (goroutines), Erlang       │
│  • User-level coroutines/fibers phổ biến trong async I/O       │
└─────────────────────────────────────────────────────────────────┘
```
