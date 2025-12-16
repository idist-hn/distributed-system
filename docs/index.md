# 📚 P2P File Sharing System - Documentation

## Tổng Quan

Hệ thống P2P File Sharing được xây dựng bằng Go, sử dụng kiến trúc hybrid P2P với tracker điều phối. Hệ thống hỗ trợ file chunking, parallel downloads, NAT traversal, và real-time monitoring.

**Live System**: https://p2p.idist.dev

## 📖 Tài Liệu

### Core Documentation

| Document | Description |
|----------|-------------|
| [Architecture](architecture.md) | Kiến trúc tổng thể hệ thống |
| [Protocol](protocol.md) | P2P protocol specification |
| [Packages](packages.md) | Chi tiết các packages |
| [Roadmap](roadmap.md) | Lộ trình phát triển |
| [Use Cases](use-cases.md) | Phân tích chi tiết use cases |
| [User Stories](user-stories.md) | User stories theo Agile format |

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
| **WebSocket Realtime** | [websocket-realtime.md](features/websocket-realtime.md) | ✅ |
| **Magnet Links** | [magnet-links.md](features/magnet-links.md) | ✅ |
| **Production Hardening** | [production-hardening.md](features/production-hardening.md) | ✅ |

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
| Packages | 12 shared packages |
| Features | 11 advanced features |
| Tests | 67+ unit tests |
| Deployment | Kubernetes (live) |
| Version | 1.3.0 |

## 🌐 Live Endpoints

| Endpoint | URL |
|----------|-----|
| Dashboard | https://p2p.idist.dev/dashboard |
| Health | https://p2p.idist.dev/health |
| Metrics | https://p2p.idist.dev/metrics |
| WebSocket | wss://p2p.idist.dev/ws |
| API | https://p2p.idist.dev/api/* |

## 📋 Analysis Documents

| Document | Description |
|----------|-------------|
| [Use Cases](use-cases.md) | 10 use cases với full analysis |
| [User Stories](user-stories.md) | 22 user stories, 80 story points |

