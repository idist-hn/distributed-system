Câu 10:
So sánh hypervisor thuần (bare metal) với hypervisor lưu trữ (hosted hypervisor) về các khía cạnh:
o	hiệu năng I/O và CPU
o	chi phí phát triển, vận hành
o	khả năng sử dụng lại trình điều khiển thiết bị (device drivers)
Trình bày ưu – nhược điểm của mỗi mô hình.
Trong bối cảnh điện toán đám mây công cộng (public cloud), hãy đánh giá mức độ phù hợp của ảo hóa dựa trên hypervisor truyền thống so với ảo hóa cấp container (container based virtualization) cho các dịch vụ:
o	(i) nền tảng PaaS (Platform as a Service) với tính năng mở rộng nhanh,
o	(ii) hạ tầng IaaS cho phép người dùng tùy chỉnh kernel.
Đưa ra khuyến nghị và giải thích trade off về bảo mật, hiệu năng, và tính di động.

---

## Trả lời:

### PHẦN 1: So sánh Bare Metal vs Hosted Hypervisor

### 1. Kiến trúc hai loại Hypervisor

```
┌─────────────────────────────────────────────────────────────────────────┐
│         TYPE 1: BARE METAL              TYPE 2: HOSTED                   │
│                                                                          │
│   ┌───────┐ ┌───────┐ ┌───────┐    ┌───────┐ ┌───────┐ ┌─────────┐    │
│   │ VM 1  │ │ VM 2  │ │ VM 3  │    │ VM 1  │ │ VM 2  │ │Host App │    │
│   │ Guest │ │ Guest │ │ Guest │    │ Guest │ │ Guest │ │         │    │
│   │  OS   │ │  OS   │ │  OS   │    │  OS   │ │  OS   │ │         │    │
│   └───┬───┘ └───┬───┘ └───┬───┘    └───┬───┘ └───┬───┘ └────┬────┘    │
│       └─────────┼─────────┘            └─────────┼──────────┘          │
│                 ↓                                ↓                      │
│   ┌─────────────────────────────┐    ┌─────────────────────────────┐  │
│   │     HYPERVISOR (Type 1)     │    │     HYPERVISOR (Type 2)     │  │
│   │  VMware ESXi, Xen, Hyper-V  │    │  VirtualBox, VMware WS      │  │
│   └─────────────┬───────────────┘    └─────────────┬───────────────┘  │
│                 ↓                                  ↓                   │
│   ┌─────────────────────────────┐    ┌─────────────────────────────┐  │
│   │       (No Host OS)          │    │       HOST OS               │  │
│   │                             │    │   Windows, Linux, macOS     │  │
│   └─────────────┬───────────────┘    └─────────────┬───────────────┘  │
│                 ↓                                  ↓                   │
│   ┌─────────────────────────────┐    ┌─────────────────────────────┐  │
│   │         HARDWARE            │    │         HARDWARE            │  │
│   └─────────────────────────────┘    └─────────────────────────────┘  │
│                                                                          │
│   Direct hardware access             Through host OS layer              │
└─────────────────────────────────────────────────────────────────────────┘
```

### 2. So sánh hiệu năng I/O và CPU

#### **2.1. Hiệu năng CPU**

| Tiêu chí | Bare Metal (Type 1) | Hosted (Type 2) |
|----------|:-------------------:|:---------------:|
| **Overhead** | ⭐ 2-5% | ⚠️ 5-15% |
| **Direct hardware access** | ✅ Có | ❌ Qua Host OS |
| **Hardware-assisted virt** | ✅ Tối ưu | ✅ Có nhưng qua layer |
| **Context switch** | Nhanh | Chậm hơn |

```
CPU EXECUTION PATH:
───────────────────

BARE METAL:
Guest App → Guest OS → Hypervisor → CPU
                         ↓
              (Direct execution với VT-x/AMD-V)

HOSTED:
Guest App → Guest OS → Hypervisor → Host OS → CPU
                                       ↓
              (Extra layer = extra overhead)
```

**Benchmark ước tính:**

| Workload | Bare Metal | Hosted | Native |
|----------|------------|--------|--------|
| CPU-intensive | 97-98% | 90-95% | 100% |
| Memory-intensive | 95-97% | 88-93% | 100% |
| Mixed | 95-98% | 85-92% | 100% |

#### **2.2. Hiệu năng I/O**

| Tiêu chí | Bare Metal (Type 1) | Hosted (Type 2) |
|----------|:-------------------:|:---------------:|
| **Disk I/O** | ⭐ 90-98% native | ⚠️ 60-85% native |
| **Network I/O** | ⭐ 95-99% native | ⚠️ 70-90% native |
| **Latency** | Thấp | Cao hơn |
| **Direct device access** | ✅ Có (SR-IOV) | ⚠️ Hạn chế |

```
I/O PATH COMPARISON:
────────────────────

BARE METAL:
┌─────────┐
│ Guest   │
│ App     │
└────┬────┘
     ↓
┌─────────┐    ┌──────────────┐
│Guest OS │ →  │ Hypervisor   │ → Hardware
│ Driver  │    │ (thin layer) │
└─────────┘    └──────────────┘

Path: 2-3 layers, LOW latency


HOSTED:
┌─────────┐
│ Guest   │
│ App     │
└────┬────┘
     ↓
┌─────────┐    ┌──────────────┐    ┌──────────┐
│Guest OS │ →  │ Hypervisor   │ →  │ Host OS  │ → Hardware
│ Driver  │    │              │    │ Driver   │
└─────────┘    └──────────────┘    └──────────┘

Path: 4-5 layers, HIGH latency
```

### 3. Chi phí phát triển và vận hành

| Tiêu chí | Bare Metal (Type 1) | Hosted (Type 2) |
|----------|:-------------------:|:---------------:|
| **Development cost** | ❌ Cao | ✅ Thấp |
| **Hardware compatibility** | ❌ Hạn chế | ✅ Rộng (dùng Host drivers) |
| **Deployment complexity** | ❌ Phức tạp | ✅ Đơn giản |
| **Management tools** | ✅ Enterprise-grade | ⚠️ Basic |
| **Licensing cost** | ❌ Cao (VMware, etc.) | ✅ Thường miễn phí |
| **Operational expertise** | ❌ Cần chuyên gia | ✅ Dễ sử dụng |
| **Scalability** | ✅ Tốt cho data center | ⚠️ Giới hạn |

**Chi phí ước tính:**

```
BARE METAL (Enterprise):
┌────────────────────────────────────────┐
│ VMware vSphere Enterprise Plus:        │
│ • License: ~$5,000/CPU                 │
│ • Support: ~$1,200/year                │
│ • Training: ~$3,000                    │
│ • Hardware certified: +20% cost        │
│ • Dedicated admin: $80,000+/year       │
│                                        │
│ TCO (3 years, 10 servers): ~$150,000+  │
└────────────────────────────────────────┘

HOSTED (Desktop/Dev):
┌────────────────────────────────────────┐
│ VirtualBox:                            │
│ • License: FREE (GPL)                  │
│ • Support: Community                   │
│ • Training: Minimal                    │
│ • Any hardware                         │
│ • No dedicated admin needed            │
│                                        │
│ TCO (3 years): ~$0 - $1,000            │
└────────────────────────────────────────┘
```

### 4. Khả năng sử dụng lại Device Drivers

| Tiêu chí | Bare Metal (Type 1) | Hosted (Type 2) |
|----------|:-------------------:|:---------------:|
| **Driver reuse** | ❌ Cần drivers riêng | ✅ Dùng Host OS drivers |
| **Hardware support** | ⚠️ Certified list only | ✅ Rất rộng |
| **Driver development** | ❌ Tốn kém | ✅ Không cần |
| **New hardware** | ⚠️ Chờ vendor support | ✅ Ngay khi Host OS support |

```
DRIVER ARCHITECTURE:
────────────────────

BARE METAL:
┌──────────────────────────────────────────────────────┐
│                   HYPERVISOR                          │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐       │
│  │ NIC Driver │ │Disk Driver │ │ GPU Driver │       │
│  │  (Custom)  │ │  (Custom)  │ │  (Custom)  │       │
│  └────────────┘ └────────────┘ └────────────┘       │
│                                                      │
│  → Phải viết/port drivers cho hypervisor            │
│  → Certified Hardware List (HCL) hạn chế            │
│  → VMware ESXi: ~2000 certified devices             │
└──────────────────────────────────────────────────────┘

HOSTED:
┌──────────────────────────────────────────────────────┐
│               HYPERVISOR (Thin)                       │
│                    ↓                                  │
│               HOST OS (Linux/Windows)                 │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐       │
│  │ NIC Driver │ │Disk Driver │ │ GPU Driver │       │
│  │  (Stock)   │ │  (Stock)   │ │  (Stock)   │       │
│  └────────────┘ └────────────┘ └────────────┘       │
│                                                      │
│  → Tận dụng tất cả drivers của Host OS              │
│  → Linux kernel: ~20,000+ device drivers            │
│  → Plug and play                                     │
└──────────────────────────────────────────────────────┘
```

### 5. Bảng tổng hợp ưu nhược điểm

#### **Bare Metal Hypervisor (Type 1):**

| Ưu điểm | Nhược điểm |
|---------|------------|
| ✅ Hiệu năng cao nhất | ❌ Chi phí cao |
| ✅ Bảo mật tốt (no Host OS) | ❌ Hardware compatibility hạn chế |
| ✅ Scalable cho data center | ❌ Phức tạp để setup |
| ✅ Enterprise management tools | ❌ Cần chuyên gia |
| ✅ Live migration, HA | ❌ Cần drivers riêng |

**Best for:** Data centers, production workloads, cloud infrastructure

#### **Hosted Hypervisor (Type 2):**

| Ưu điểm | Nhược điểm |
|---------|------------|
| ✅ Dễ cài đặt và sử dụng | ❌ Hiệu năng thấp hơn |
| ✅ Chi phí thấp/miễn phí | ❌ Host OS overhead |
| ✅ Hardware compatibility rộng | ❌ Bảo mật yếu hơn |
| ✅ Tận dụng Host OS drivers | ❌ Không phù hợp production |
| ✅ Chạy song song với Host apps | ❌ Scalability hạn chế |

**Best for:** Development, testing, desktop virtualization, learning

---

### PHẦN 2: Hypervisor vs Container trong Public Cloud

### 6. So sánh Hypervisor và Container

```
┌─────────────────────────────────────────────────────────────────────────┐
│           HYPERVISOR VM                    CONTAINER                     │
│                                                                          │
│   ┌───────┐ ┌───────┐ ┌───────┐    ┌───────┐ ┌───────┐ ┌───────┐      │
│   │ App A │ │ App B │ │ App C │    │ App A │ │ App B │ │ App C │      │
│   ├───────┤ ├───────┤ ├───────┤    ├───────┤ ├───────┤ ├───────┤      │
│   │ Bins/ │ │ Bins/ │ │ Bins/ │    │ Bins/ │ │ Bins/ │ │ Bins/ │      │
│   │ Libs  │ │ Libs  │ │ Libs  │    │ Libs  │ │ Libs  │ │ Libs  │      │
│   ├───────┤ ├───────┤ ├───────┤    └───┬───┘ └───┬───┘ └───┬───┘      │
│   │Guest  │ │Guest  │ │Guest  │        └─────────┼─────────┘           │
│   │  OS   │ │  OS   │ │  OS   │                  ↓                      │
│   └───┬───┘ └───┬───┘ └───┬───┘    ┌─────────────────────────────┐    │
│       └─────────┼─────────┘        │     Container Runtime        │    │
│                 ↓                   │   (Docker, containerd)      │    │
│   ┌─────────────────────────────┐  └─────────────┬───────────────┘    │
│   │         HYPERVISOR          │                ↓                     │
│   └─────────────┬───────────────┘  ┌─────────────────────────────┐    │
│                 ↓                   │         HOST OS              │    │
│   ┌─────────────────────────────┐  │   (Shared Kernel)           │    │
│   │         HARDWARE            │  └─────────────┬───────────────┘    │
│   └─────────────────────────────┘                ↓                     │
│                                    ┌─────────────────────────────┐    │
│   Full OS per VM                   │         HARDWARE            │    │
│   Strong isolation                 └─────────────────────────────┘    │
│   Heavy (GBs)                                                          │
│                                    Shared kernel                        │
│                                    Lighter isolation                   │
│                                    Light (MBs)                          │
└─────────────────────────────────────────────────────────────────────────┘
```

#### **Bảng so sánh chi tiết:**

| Tiêu chí | Hypervisor VM | Container |
|----------|:-------------:|:---------:|
| **Isolation** | ⭐⭐⭐⭐⭐ Complete | ⭐⭐⭐ Process-level |
| **Startup time** | 30s - 5min | ⭐ 100ms - 1s |
| **Resource overhead** | 500MB - GBs | ⭐ 10-100 MBs |
| **Density** | 10s VMs/server | ⭐ 100s-1000s/server |
| **Portability** | ⚠️ VM images (large) | ⭐ Container images (small) |
| **Kernel customization** | ⭐ Yes (own kernel) | ❌ No (shared kernel) |
| **Security** | ⭐⭐⭐⭐⭐ Strong | ⭐⭐⭐ Good (improving) |
| **Performance** | 95-98% native | ⭐ 99%+ native |

### 7. Đánh giá cho từng use case

#### **7.1. PaaS với tính năng mở rộng nhanh**

**Yêu cầu PaaS:**
- ⚡ Scale up/down nhanh (seconds)
- 📦 Deploy ứng dụng thường xuyên
- 💰 Chi phí hiệu quả (high density)
- 🔄 CI/CD integration
- 📊 Resource efficiency

```
PAAS SCALING SCENARIO:
──────────────────────

Traffic Spike: 100 → 1000 requests/sec

HYPERVISOR VM:
┌────────────────────────────────────────────────────────┐
│ T=0:  [VM1] [VM2] [VM3]                               │
│ T=30s: Starting new VMs...                             │
│ T=60s: [VM1] [VM2] [VM3] [VM4] [VM5]                  │
│ T=90s: [VM1] [VM2] [VM3] [VM4] [VM5] [VM6] [VM7]      │
│                                                        │
│ ⚠️ Scaling time: 30-60 seconds per VM                 │
│ ⚠️ During scaling: requests may timeout/drop          │
└────────────────────────────────────────────────────────┘

CONTAINER:
┌────────────────────────────────────────────────────────┐
│ T=0:    [C1] [C2] [C3]                                 │
│ T=1s:   [C1] [C2] [C3] [C4] [C5] [C6] [C7]            │
│ T=2s:   [C1] [C2] [C3] [C4] [C5] [C6] [C7] [C8]...[C20]│
│                                                        │
│ ✅ Scaling time: <1 second per container              │
│ ✅ Near-instant response to traffic                   │
└────────────────────────────────────────────────────────┘
```

**Đánh giá cho PaaS:**

| Tiêu chí | Hypervisor | Container | Winner |
|----------|:----------:|:---------:|:------:|
| Scale speed | ⭐⭐ | ⭐⭐⭐⭐⭐ | Container |
| Density | ⭐⭐ | ⭐⭐⭐⭐⭐ | Container |
| Deployment | ⭐⭐ | ⭐⭐⭐⭐⭐ | Container |
| Resource efficiency | ⭐⭐ | ⭐⭐⭐⭐⭐ | Container |
| CI/CD integration | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | Container |

**📌 Khuyến nghị: CONTAINER cho PaaS**

#### **7.2. IaaS cho phép người dùng tùy chỉnh kernel**

**Yêu cầu IaaS:**
- 🔧 Custom kernel (versions, modules, patches)
- 🛡️ Strong isolation giữa tenants
- 💻 Full OS control cho user
- 🔒 Security compliance
- 🖥️ Multiple OS support (Linux, Windows)

```
KERNEL CUSTOMIZATION SCENARIO:
──────────────────────────────

User A: Needs Linux 5.15 with custom I/O scheduler
User B: Needs Windows Server 2022
User C: Needs Linux 4.19 LTS with real-time patches

HYPERVISOR VM:
┌────────────────────────────────────────────────────────┐
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │   User A    │  │   User B    │  │   User C    │    │
│  │ Linux 5.15  │  │ Windows     │  │ Linux 4.19  │    │
│  │ Custom I/O  │  │ Server 2022 │  │ RT patches  │    │
│  │   Kernel    │  │   Kernel    │  │   Kernel    │    │
│  └─────────────┘  └─────────────┘  └─────────────┘    │
│                                                        │
│  ✅ Each VM has its own kernel                        │
│  ✅ Full customization possible                       │
│  ✅ Any OS supported                                  │
└────────────────────────────────────────────────────────┘

CONTAINER:
┌────────────────────────────────────────────────────────┐
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │   User A    │  │   User B    │  │   User C    │    │
│  │   App       │  │   App       │  │   App       │    │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘    │
│         └────────────────┼────────────────┘            │
│                          ↓                             │
│              ┌──────────────────────┐                 │
│              │   SHARED KERNEL      │                 │
│              │   (Host's kernel)    │                 │
│              └──────────────────────┘                 │
│                                                        │
│  ❌ Cannot customize kernel per user                  │
│  ❌ Windows containers need Windows host              │
│  ❌ Kernel modules shared                             │
└────────────────────────────────────────────────────────┘
```

**Đánh giá cho IaaS (kernel customization):**

| Tiêu chí | Hypervisor | Container | Winner |
|----------|:----------:|:---------:|:------:|
| Custom kernel | ⭐⭐⭐⭐⭐ | ❌ | Hypervisor |
| Multi-OS | ⭐⭐⭐⭐⭐ | ⭐⭐ | Hypervisor |
| Tenant isolation | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | Hypervisor |
| Kernel modules | ⭐⭐⭐⭐⭐ | ⭐ | Hypervisor |
| Compliance (SOC2, etc.) | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | Hypervisor |

**📌 Khuyến nghị: HYPERVISOR VM cho IaaS với kernel customization**

### 8. Trade-offs: Bảo mật, Hiệu năng, Tính di động

#### **8.1. Bảo mật (Security)**

```
SECURITY COMPARISON:
────────────────────

HYPERVISOR:
┌─────────────────────────────────────────────────────┐
│ Attack Surface:                                      │
│ • Hypervisor + Guest OS                             │
│ • Hardware-level isolation (VT-x)                   │
│ • Separate kernel per VM                            │
│                                                      │
│ Threats:                                             │
│ • VM escape (rare, critical)                        │
│ • Side-channel attacks (Spectre, Meltdown)          │
│                                                      │
│ Security Level: ⭐⭐⭐⭐⭐ (Industry standard)        │
└─────────────────────────────────────────────────────┘

CONTAINER:
┌─────────────────────────────────────────────────────┐
│ Attack Surface:                                      │
│ • Container runtime + Shared kernel                 │
│ • Namespace/cgroup isolation (software)             │
│ • Shared kernel = shared vulnerabilities            │
│                                                      │
│ Threats:                                             │
│ • Container escape (more common than VM escape)     │
│ • Kernel exploits affect all containers             │
│ • Privileged containers = root on host              │
│                                                      │
│ Security Level: ⭐⭐⭐ (Improving with gVisor, Kata) │
└─────────────────────────────────────────────────────┘
```

| Security Aspect | Hypervisor | Container |
|-----------------|:----------:|:---------:|
| Isolation strength | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| Kernel vulnerabilities | Isolated | Shared risk |
| Escape difficulty | Very hard | Easier |
| Multi-tenant trust | High | Medium |
| Compliance ready | Yes | Depends |

#### **8.2. Hiệu năng (Performance)**

```
PERFORMANCE COMPARISON:
───────────────────────

                    Hypervisor      Container
CPU Performance:    95-98%          99-100%
Memory Overhead:    500MB-2GB/VM    10-50MB/container
Startup Time:       30-120 sec      0.1-1 sec
I/O Performance:    90-95%          98-100%
Network Latency:    +50-100μs       +10-20μs
Density:            10-50 VMs       100-1000 containers
```

| Performance Aspect | Hypervisor | Container | Difference |
|-------------------|:----------:|:---------:|:----------:|
| CPU overhead | 2-5% | <1% | Container +4% |
| Memory per instance | 512MB+ | 10MB+ | Container 50x better |
| Startup | 30-120s | <1s | Container 100x faster |
| I/O throughput | 90-95% | 98%+ | Container +5% |

#### **8.3. Tính di động (Portability)**

```
PORTABILITY COMPARISON:
───────────────────────

HYPERVISOR:
┌─────────────────────────────────────────────────────┐
│ VM Image:                                            │
│ • Size: 10-100 GB                                   │
│ • Format: VMDK, VHD, QCOW2 (not standardized)       │
│ • Contains: Full OS + Apps                          │
│ • Transfer time: Minutes to hours                   │
│ • Cross-platform: Limited (hypervisor-specific)     │
│                                                      │
│ Migration: OVF format helps, but still complex      │
└─────────────────────────────────────────────────────┘

CONTAINER:
┌─────────────────────────────────────────────────────┐
│ Container Image:                                     │
│ • Size: 10-500 MB (layered)                         │
│ • Format: OCI standard (universal)                  │
│ • Contains: App + dependencies only                 │
│ • Transfer time: Seconds to minutes                 │
│ • Cross-platform: Any Linux host (mostly)           │
│                                                      │
│ Migration: docker push/pull anywhere                │
└─────────────────────────────────────────────────────┘
```

| Portability Aspect | Hypervisor | Container |
|-------------------|:----------:|:---------:|
| Image size | 10-100 GB | 10-500 MB |
| Standardization | ⚠️ OVF/OVA | ✅ OCI |
| Registry ecosystem | Limited | Docker Hub, etc. |
| Build reproducibility | Harder | Dockerfile |
| Cross-cloud | Complex | Easy |

### 9. Bảng tổng hợp khuyến nghị

| Use Case | Khuyến nghị | Lý do |
|----------|-------------|-------|
| **PaaS (fast scaling)** | ✅ **Container** | Startup <1s, high density, CI/CD friendly |
| **IaaS (custom kernel)** | ✅ **Hypervisor** | Full OS control, strong isolation |
| **Multi-tenant SaaS** | Hypervisor hoặc Kata Containers | Security isolation critical |
| **Microservices** | ✅ **Container** | Lightweight, scalable |
| **Legacy Windows apps** | ✅ **Hypervisor** | Full Windows support |
| **Development/Testing** | ✅ **Container** | Fast, reproducible |

### 10. Kết luận

```
┌─────────────────────────────────────────────────────────────────────┐
│                           SUMMARY                                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  BARE METAL vs HOSTED HYPERVISOR:                                   │
│  ├─ Bare Metal: Production, performance, enterprise                │
│  └─ Hosted: Development, testing, desktop                          │
│                                                                      │
│  HYPERVISOR vs CONTAINER (Public Cloud):                            │
│  ├─ PaaS (fast scaling): → CONTAINER ✅                             │
│  │   • Startup <1s, high density, perfect for auto-scaling         │
│  │                                                                   │
│  └─ IaaS (custom kernel): → HYPERVISOR ✅                           │
│      • Full kernel control, strong isolation, compliance           │
│                                                                      │
│  TRADE-OFFS:                                                         │
│  ├─ Security:    Hypervisor > Container                             │
│  ├─ Performance: Container > Hypervisor                             │
│  └─ Portability: Container > Hypervisor                             │
│                                                                      │
│  HYBRID APPROACH (Best practice):                                   │
│  ├─ Run containers inside VMs for security                         │
│  ├─ Example: GKE, EKS run on VMs underneath                        │
│  └─ Kata Containers: VM-level isolation + container UX             │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```
