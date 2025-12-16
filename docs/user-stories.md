# 📖 User Stories - P2P File Sharing System

## Tổng Quan

Tài liệu này mô tả đầy đủ các User Stories theo format Agile, bao gồm acceptance criteria và technical notes.

---

## Epic 1: Peer Management

### US-1.1: Đăng Ký Peer
**As a** peer node  
**I want to** register with the tracker  
**So that** I can participate in the P2P network

**Acceptance Criteria:**
- [ ] Peer có thể gửi registration request với peer_id, ip, port
- [ ] Tracker xác thực API key
- [ ] Tracker lưu peer info vào database
- [ ] Tracker trả về success response
- [ ] Dashboard cập nhật realtime khi có peer mới

**Priority:** High | **Story Points:** 3

---

### US-1.2: Heartbeat
**As a** registered peer  
**I want to** send periodic heartbeats  
**So that** the tracker knows I'm still online

**Acceptance Criteria:**
- [ ] Peer gửi heartbeat mỗi 30 giây
- [ ] Tracker cập nhật last_seen timestamp
- [ ] Peer offline sau 90 giây không heartbeat
- [ ] Dashboard hiển thị trạng thái online/offline

**Priority:** High | **Story Points:** 2

---

### US-1.3: Peer Discovery
**As a** peer  
**I want to** discover other peers sharing a file  
**So that** I can connect and download

**Acceptance Criteria:**
- [ ] API trả về danh sách peers có file
- [ ] Danh sách bao gồm IP, port, peer_id
- [ ] Peers được sắp xếp theo score
- [ ] Chỉ trả về peers online

**Priority:** High | **Story Points:** 2

---

## Epic 2: File Sharing

### US-2.1: Chia Sẻ File
**As a** seeder  
**I want to** announce that I have a file  
**So that** other peers can find and download it

**Acceptance Criteria:**
- [ ] File được chunk thành 256KB pieces
- [ ] Mỗi chunk có SHA-256 hash
- [ ] Metadata được gửi đến tracker
- [ ] File xuất hiện trong search results
- [ ] Magnet link có thể generate

**Priority:** High | **Story Points:** 5

---

### US-2.2: Tìm Kiếm File
**As a** leecher  
**I want to** search for files by name  
**So that** I can find content to download

**Acceptance Criteria:**
- [ ] Full-text search hoạt động
- [ ] Kết quả bao gồm name, size, hash, peer count
- [ ] Search by hash cũng hoạt động
- [ ] Response time < 200ms

**Priority:** High | **Story Points:** 3

---

### US-2.3: Browse Categories
**As a** user  
**I want to** browse files by category  
**So that** I can discover new content

**Acceptance Criteria:**
- [ ] API trả về danh sách categories
- [ ] Mỗi category có file count
- [ ] Có thể filter files theo category

**Priority:** Medium | **Story Points:** 2

---

## Epic 3: File Download

### US-3.1: Download File
**As a** leecher  
**I want to** download a file from peers  
**So that** I can get the content I need

**Acceptance Criteria:**
- [ ] Kết nối TCP trực tiếp đến seeder
- [ ] Handshake protocol hoạt động
- [ ] Chunks được download song song
- [ ] Mỗi chunk được verify hash
- [ ] File hoàn chỉnh được assemble

**Priority:** High | **Story Points:** 8

---

### US-3.2: Parallel Download
**As a** leecher  
**I want to** download chunks from multiple peers simultaneously  
**So that** download speed is maximized

**Acceptance Criteria:**
- [ ] Có thể connect đến nhiều peers cùng lúc
- [ ] Mỗi peer download các chunks khác nhau
- [ ] Worker pool quản lý connections
- [ ] Download speed tăng tuyến tính với số peers

**Priority:** High | **Story Points:** 5

---

### US-3.3: Resume Download
**As a** leecher  
**I want to** resume interrupted downloads  
**So that** I don't have to start over

**Acceptance Criteria:**
- [ ] Progress được lưu vào .progress file
- [ ] Khi restart, đọc progress file
- [ ] Chỉ download chunks còn thiếu
- [ ] Hash verification cho chunks đã có

**Priority:** High | **Story Points:** 5

---

### US-3.4: Pause Download
**As a** leecher  
**I want to** pause and resume downloads  
**So that** I can control bandwidth usage

**Acceptance Criteria:**
- [ ] Pause button dừng download
- [ ] Progress được lưu
- [ ] Resume button tiếp tục
- [ ] Không mất data đã download

**Priority:** Medium | **Story Points:** 3

---

### US-3.5: Endgame Mode
**As a** leecher  
**I want to** quickly finish downloads  
**So that** the last few chunks don't take forever

**Acceptance Criteria:**
- [ ] Khi còn < 5% chunks → activate endgame
- [ ] Request remaining chunks từ tất cả peers
- [ ] Cancel duplicate khi chunk received
- [ ] Download hoàn thành nhanh hơn

**Priority:** Medium | **Story Points:** 3

---

## Epic 4: NAT Traversal

### US-4.1: Direct Connection
**As a** peer  
**I want to** connect directly to another peer  
**So that** transfer is fast and efficient

**Acceptance Criteria:**
- [ ] TCP connection đến IP:port
- [ ] Timeout 5 giây nếu fail
- [ ] Fallback to hole punching

**Priority:** High | **Story Points:** 2

---

### US-4.2: NAT Hole Punching
**As a** peer behind NAT  
**I want to** connect to another NATed peer  
**So that** I can download without relay

**Acceptance Criteria:**
- [ ] Coordinate qua tracker
- [ ] Simultaneous UDP packets
- [ ] 3 retry attempts
- [ ] Fallback to relay nếu fail

**Priority:** High | **Story Points:** 8

---

### US-4.3: Relay Connection
**As a** peer  
**I want to** relay through tracker  
**So that** I can connect when direct/punch fails

**Acceptance Criteria:**
- [ ] WebSocket connection đến /relay
- [ ] Tracker relay messages
- [ ] Works với symmetric NAT
- [ ] Bandwidth limited

**Priority:** High | **Story Points:** 5

---

## Epic 5: Security

### US-5.1: API Authentication
**As a** system admin  
**I want to** require API keys  
**So that** only authorized peers can use the system

**Acceptance Criteria:**
- [ ] API key required trong header
- [ ] Invalid key → 401 response
- [ ] Multiple API keys supported
- [ ] Keys loaded từ environment

**Priority:** High | **Story Points:** 3

---

### US-5.2: Rate Limiting
**As a** system admin  
**I want to** limit API requests  
**So that** the system isn't overwhelmed

**Acceptance Criteria:**
- [ ] 100 requests/minute per IP
- [ ] 429 response khi exceeded
- [ ] Token bucket algorithm
- [ ] Health endpoint exempt

**Priority:** High | **Story Points:** 3

---

### US-5.3: End-to-End Encryption
**As a** user  
**I want to** encrypt file transfers  
**So that** my data is private

**Acceptance Criteria:**
- [ ] AES-256-GCM encryption
- [ ] Key exchange via ECDH
- [ ] Per-chunk encryption
- [ ] No plaintext on wire

**Priority:** Medium | **Story Points:** 5

---

## Epic 6: Monitoring

### US-6.1: Web Dashboard
**As a** admin/user  
**I want to** see system status  
**So that** I can monitor the network

**Acceptance Criteria:**
- [ ] Stats cards: peers, files, relay, status
- [ ] Peers table với details
- [ ] Files table với details
- [ ] Real-time updates via WebSocket

**Priority:** High | **Story Points:** 5

---

### US-6.2: WebSocket Realtime
**As a** dashboard user  
**I want to** see real-time updates  
**So that** I don't need to refresh

**Acceptance Criteria:**
- [ ] WebSocket auto-connect
- [ ] Connection status indicator
- [ ] Toast notifications cho events
- [ ] Stats update mỗi 5 giây
- [ ] Auto-reconnect với backoff

**Priority:** High | **Story Points:** 5

---

### US-6.3: Prometheus Metrics
**As a** DevOps engineer  
**I want to** export metrics  
**So that** I can monitor with Grafana

**Acceptance Criteria:**
- [ ] /metrics endpoint
- [ ] HTTP request metrics
- [ ] Peer count gauge
- [ ] File count gauge
- [ ] Histogram cho latency

**Priority:** Medium | **Story Points:** 3

---

## Epic 7: Content Discovery

### US-7.1: Magnet Links
**As a** seeder  
**I want to** generate magnet links  
**So that** I can share files easily

**Acceptance Criteria:**
- [ ] API generate magnet URI
- [ ] URI contains hash, name, size
- [ ] URI contains tracker URL
- [ ] Magnet link có thể parse

**Priority:** Medium | **Story Points:** 3

---

### US-7.2: Parse Magnet Link
**As a** leecher  
**I want to** paste a magnet link  
**So that** I can start downloading

**Acceptance Criteria:**
- [ ] API parse magnet URI
- [ ] Extract hash, name, size
- [ ] Lookup file in tracker
- [ ] Return file info

**Priority:** Medium | **Story Points:** 2

---

## Story Map

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              USER JOURNEY                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Register ──▶ Announce ──▶ Share Link ──▶ Search ──▶ Download ──▶ Seed     │
│     │            │            │             │           │           │        │
│  US-1.1       US-2.1       US-7.1        US-2.2      US-3.1      US-2.1     │
│  US-1.2                    US-7.2        US-2.3      US-3.2                  │
│  US-1.3                                              US-3.3                  │
│                                                      US-3.4                  │
│                                                      US-3.5                  │
│                                                                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                           INFRASTRUCTURE                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  NAT Traversal          Security              Monitoring                     │
│  ─────────────          ────────              ──────────                     │
│  US-4.1                 US-5.1                US-6.1                         │
│  US-4.2                 US-5.2                US-6.2                         │
│  US-4.3                 US-5.3                US-6.3                         │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Priority Summary

| Priority | Stories | Total Points |
|----------|---------|--------------|
| High | 14 | 54 |
| Medium | 8 | 26 |
| **Total** | **22** | **80** |

---

## Sprint Planning Suggestion

### Sprint 1: Core (2 weeks)
- US-1.1, US-1.2, US-1.3 (7 pts)
- US-2.1, US-2.2 (8 pts)
- US-3.1 (8 pts)
**Total: 23 points**

### Sprint 2: Download Features (2 weeks)
- US-3.2, US-3.3, US-3.4, US-3.5 (16 pts)
- US-4.1 (2 pts)
**Total: 18 points**

### Sprint 3: NAT & Security (2 weeks)
- US-4.2, US-4.3 (13 pts)
- US-5.1, US-5.2 (6 pts)
**Total: 19 points**

### Sprint 4: Monitoring & Polish (2 weeks)
- US-6.1, US-6.2, US-6.3 (13 pts)
- US-7.1, US-7.2 (5 pts)
- US-5.3, US-2.3 (7 pts)
**Total: 25 points**

