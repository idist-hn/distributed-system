# Magnet Links

## Tổng quan

**Magnet Links** cho phép chia sẻ files mà không cần host file trực tiếp. Chỉ cần share một URI text chứa đủ thông tin để tìm và download file.

## Magnet URI Format

```
magnet:?xt=urn:sha256:<hash>&dn=<name>&xl=<size>&tr=<tracker_url>
```

### Parameters

| Parameter | Mô tả | Required |
|-----------|-------|----------|
| `xt` | eXact Topic - Hash của file (urn:sha256:xxx) | ✅ |
| `dn` | Display Name - Tên file | ❌ |
| `xl` | eXact Length - Kích thước file (bytes) | ❌ |
| `tr` | TRacker URL - URL của tracker | ❌ |

### Ví dụ

```
magnet:?xt=urn:sha256:abc123def456...&dn=movie.mp4&xl=1073741824&tr=https://p2p.idist.dev
```

## API Endpoints

### Generate Magnet Link

```http
GET /api/files/{hash}/magnet
X-API-Key: peer-key-001
```

**Response:**
```json
{
  "magnet": "magnet:?xt=urn:sha256:abc123...&dn=movie.mp4&xl=1073741824&tr=https://p2p.idist.dev",
  "hash": "abc123...",
  "name": "movie.mp4",
  "size": 1073741824
}
```

### Parse Magnet Link

```http
GET /api/magnet?uri=magnet:?xt=urn:sha256:abc123...
X-API-Key: peer-key-001
```

**Response:**
```json
{
  "hash": "abc123def456...",
  "name": "movie.mp4",
  "size": 1073741824,
  "trackers": ["https://p2p.idist.dev"],
  "file_found": true,
  "peer_count": 5
}
```

## Package Implementation

### Location
```
pkg/magnet/magnet.go
```

### Core Functions

```go
// Generate magnet URI từ file info
func Generate(hash, name string, size int64, trackers []string) string

// Parse magnet URI thành struct
func Parse(uri string) (*MagnetInfo, error)

// Validate magnet URI format
func IsValid(uri string) bool
```

### MagnetInfo Struct

```go
type MagnetInfo struct {
    Hash     string   `json:"hash"`
    Name     string   `json:"name"`
    Size     int64    `json:"size"`
    Trackers []string `json:"trackers"`
}
```

## Usage Flow

### Sharing a File

```
┌─────────┐                    ┌─────────┐                    ┌─────────┐
│ Seeder  │                    │ Tracker │                    │ Leecher │
└────┬────┘                    └────┬────┘                    └────┬────┘
     │                              │                              │
     │── Announce file ────────────▶│                              │
     │                              │                              │
     │── GET /files/{hash}/magnet ─▶│                              │
     │◀── Magnet URI ───────────────│                              │
     │                              │                              │
     │                              │                              │
     │════════════════════════════════════════════════════════════│
     │         Share magnet link (email, chat, website)            │
     │════════════════════════════════════════════════════════════│
     │                              │                              │
     │                              │◀── GET /api/magnet?uri= ─────│
     │                              │── File info ────────────────▶│
     │                              │                              │
     │                              │◀── GET /files/{hash}/peers ──│
     │                              │── Peer list ────────────────▶│
     │                              │                              │
     │◀═══════════════════════════════════ P2P Download ══════════│
     │                              │                              │
```

### Downloading via Magnet

1. User paste magnet link
2. Client calls `GET /api/magnet?uri=...`
3. Tracker returns file info & peer count
4. Client calls `GET /api/files/{hash}/peers`
5. Client connects to peers và download

## Code Examples

### Generate Magnet (Go)

```go
import "p2p-system/pkg/magnet"

// Generate magnet link
link := magnet.Generate(
    "abc123def456...",           // hash
    "movie.mp4",                  // name
    1073741824,                   // size
    []string{"https://p2p.idist.dev"}, // trackers
)
// Result: magnet:?xt=urn:sha256:abc123...&dn=movie.mp4&xl=1073741824&tr=https://p2p.idist.dev
```

### Parse Magnet (Go)

```go
import "p2p-system/pkg/magnet"

info, err := magnet.Parse("magnet:?xt=urn:sha256:abc123...")
if err != nil {
    log.Fatal(err)
}

fmt.Println(info.Hash)     // abc123...
fmt.Println(info.Name)     // movie.mp4
fmt.Println(info.Size)     // 1073741824
fmt.Println(info.Trackers) // [https://p2p.idist.dev]
```

### Parse Magnet (JavaScript)

```javascript
async function downloadFromMagnet(magnetUri) {
    // Parse magnet
    const response = await fetch(`/api/magnet?uri=${encodeURIComponent(magnetUri)}`, {
        headers: { 'X-API-Key': 'peer-key-001' }
    });
    const info = await response.json();
    
    if (!info.file_found) {
        throw new Error('File not found in tracker');
    }
    
    // Get peers and start download
    const peers = await fetch(`/api/files/${info.hash}/peers`);
    // ... start P2P download
}
```

## Tests

```bash
go test ./pkg/magnet/... -v
```

| Test | Description |
|------|-------------|
| TestGenerate | Generate magnet với đầy đủ params |
| TestGenerateMinimal | Generate với chỉ hash |
| TestParse | Parse magnet URI đầy đủ |
| TestParseMinimal | Parse magnet chỉ có hash |
| TestParseInvalid | Handle invalid URI |
| TestIsValid | Validate magnet format |
| TestURLEncoding | Handle special characters |
| TestMultipleTrackers | Multiple tracker URLs |

## Lưu ý

1. Hash type mặc định là `sha256` (không phải `btih` như BitTorrent)
2. Name và special characters được URL-encoded
3. Multiple trackers được hỗ trợ với nhiều `&tr=` params
4. Magnet link có thể share qua bất kỳ medium nào (text-based)

