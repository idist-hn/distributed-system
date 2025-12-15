# 📚 P2P File Sharing System - Documentation

## Tổng Quan

Hệ thống P2P File Sharing được xây dựng bằng Go, sử dụng kiến trúc hybrid P2P với tracker điều phối.

## 📖 Tài Liệu

### Core Documentation

| Document | Description |
|----------|-------------|
| [Architecture](architecture.md) | Kiến trúc tổng thể hệ thống |
| [Protocol](protocol.md) | P2P protocol specification |
| [Packages](packages.md) | Chi tiết các packages |
| [Roadmap](roadmap.md) | Lộ trình phát triển |

### API Reference

| Document | Description |
|----------|-------------|
| [Postman Collection](P2P-Tracker-API.postman_collection.json) | API collection for testing |

### Feature Documentation

| Feature | Document | Status |
|---------|----------|--------|
| Parallel Chunk Downloads | [parallel-chunk-downloads.md](features/parallel-chunk-downloads.md) | ✅ |
| Resume/Pause Downloads | [resume-pause-downloads.md](features/resume-pause-downloads.md) | ✅ |
| End-to-End Encryption | [end-to-end-encryption.md](features/end-to-end-encryption.md) | ✅ |
| DHT Kademlia | [dht-kademlia.md](features/dht-kademlia.md) | ✅ |
| Web UI Dashboard | [web-ui-dashboard.md](features/web-ui-dashboard.md) | ✅ |
| Bandwidth Throttling | [bandwidth-throttling.md](features/bandwidth-throttling.md) | ✅ |
| Merkle Tree Verification | [merkle-tree-verification.md](features/merkle-tree-verification.md) | ✅ |
| NAT Hole Punching | [nat-hole-punching.md](features/nat-hole-punching.md) | ✅ |

## 🏗️ Kiến Trúc

```
┌─────────────────────────────────────────────────────────────┐
│                       TRACKER                                │
│   REST API • WebSocket • Dashboard • Hole Punch Coordinator │
└─────────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
   ┌─────────┐         ┌─────────┐         ┌─────────┐
   │ PEER A  │◄───────▶│ PEER B  │◄───────▶│ PEER C  │
   └─────────┘         └─────────┘         └─────────┘
```

## 🔧 Components

### Tracker Server (`services/tracker`)
- REST API for peer/file management
- WebSocket for real-time events
- Relay hub for NAT traversal
- Web dashboard for monitoring

### Peer Node (`services/peer`)
- P2P TCP server for file transfer
- Connection manager (Direct → Punch → Relay)
- Parallel chunk downloader
- Local storage with resume capability

### Shared Packages (`pkg/`)
- `chunker` - File chunking
- `crypto` - E2E encryption
- `dht` - Kademlia DHT
- `hash` - SHA-256 hashing
- `holepunch` - NAT hole punching
- `merkle` - Merkle tree
- `protocol` - Message definitions
- `throttle` - Bandwidth limiting

## 🚀 Quick Links

- [README](../README.md) - Getting started
- [Architecture](architecture.md) - System design
- [Roadmap](roadmap.md) - Future development
- [Packages](packages.md) - Package reference

## 📊 System Stats

| Metric | Value |
|--------|-------|
| Language | Go 1.21+ |
| Packages | 9 shared packages |
| Features | 8 advanced features |
| Tests | 35+ unit tests |
| Deployment | Kubernetes ready |

