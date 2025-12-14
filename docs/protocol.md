# Protocol Specification

## 1. Tracker API (REST/gRPC)

### 1.1 Peer Registration

**Endpoint**: `POST /api/peers/register`

```json
// Request
{
  "peer_id": "uuid-string",
  "ip": "192.168.1.10",
  "port": 6881,
  "hostname": "peer-node-1"
}

// Response
{
  "success": true,
  "message": "Registered successfully",
  "session_token": "jwt-token-here"
}
```

### 1.2 Peer Heartbeat

**Endpoint**: `POST /api/peers/heartbeat`

```json
// Request
{
  "peer_id": "uuid-string",
  "files_sharing": ["file_hash_1", "file_hash_2"]
}

// Response
{
  "success": true,
  "next_heartbeat_in": 30
}
```

### 1.3 Announce File

**Endpoint**: `POST /api/files/announce`

```json
// Request
{
  "peer_id": "uuid-string",
  "file": {
    "name": "video.mp4",
    "size": 104857600,
    "hash": "sha256:abc123def456...",
    "chunk_size": 262144,
    "chunks": [
      {"index": 0, "hash": "sha256:chunk0hash...", "size": 262144},
      {"index": 1, "hash": "sha256:chunk1hash...", "size": 262144}
    ]
  }
}

// Response
{
  "success": true,
  "file_id": "file-uuid"
}
```

### 1.4 Get Peers for File

**Endpoint**: `GET /api/files/{file_hash}/peers`

```json
// Response
{
  "file_hash": "sha256:abc123def456...",
  "file_name": "video.mp4",
  "file_size": 104857600,
  "chunk_count": 400,
  "peers": [
    {
      "peer_id": "peer-1",
      "ip": "192.168.1.10",
      "port": 6881,
      "chunks_available": [0, 1, 2, 3, ...399],
      "is_seeder": true
    },
    {
      "peer_id": "peer-2",
      "ip": "192.168.1.11",
      "port": 6881,
      "chunks_available": [0, 1, 2],
      "is_seeder": false
    }
  ]
}
```

### 1.5 List Available Files

**Endpoint**: `GET /api/files`

```json
// Response
{
  "files": [
    {
      "hash": "sha256:abc123...",
      "name": "video.mp4",
      "size": 104857600,
      "seeders": 5,
      "leechers": 2
    }
  ]
}
```

## 2. Peer-to-Peer Protocol (TCP)

### 2.1 Handshake

```json
{
  "type": "HANDSHAKE",
  "peer_id": "uuid-string",
  "version": "1.0"
}
```

### 2.2 Request Chunk

```json
{
  "type": "REQUEST_CHUNK",
  "file_hash": "sha256:abc123...",
  "chunk_index": 5
}
```

### 2.3 Chunk Response

```json
{
  "type": "CHUNK_DATA",
  "file_hash": "sha256:abc123...",
  "chunk_index": 5,
  "chunk_hash": "sha256:chunk5hash...",
  "data": "<base64-encoded-chunk-data>"
}
```

### 2.4 Have (Thông báo có chunk)

```json
{
  "type": "HAVE",
  "file_hash": "sha256:abc123...",
  "chunk_index": 5
}
```

### 2.5 Bitfield (Danh sách chunks đang có)

```json
{
  "type": "BITFIELD",
  "file_hash": "sha256:abc123...",
  "bitfield": "11111111110000001111..."
}
```

## 3. Error Codes

| Code | Message | Description |
|------|---------|-------------|
| 1001 | PEER_NOT_FOUND | Peer không tồn tại |
| 1002 | FILE_NOT_FOUND | File không tồn tại |
| 1003 | CHUNK_NOT_AVAILABLE | Peer không có chunk này |
| 1004 | HASH_MISMATCH | Hash không khớp |
| 1005 | CONNECTION_REFUSED | Từ chối kết nối |

