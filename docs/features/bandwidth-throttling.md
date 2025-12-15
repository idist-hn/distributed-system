# Bandwidth Throttling

## Tổng quan

**Bandwidth Throttling** giới hạn tốc độ upload/download để kiểm soát băng thông sử dụng.

## Kiến trúc

```
┌─────────────────────────────────────────────────────────────────┐
│                   BANDWIDTH THROTTLING                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │                  BandwidthManager                        │   │
│   │                                                          │   │
│   │  ┌──────────────┐          ┌──────────────┐             │   │
│   │  │Upload Limiter│          │Download Limit│             │   │
│   │  │  (Token      │          │  (Token      │             │   │
│   │  │   Bucket)    │          │   Bucket)    │             │   │
│   │  └──────┬───────┘          └──────┬───────┘             │   │
│   │         │                         │                      │   │
│   │         ▼                         ▼                      │   │
│   │  ┌──────────────┐          ┌──────────────┐             │   │
│   │  │ThrottledWrite│          │ThrottledReade│             │   │
│   │  └──────────────┘          └──────────────┘             │   │
│   │                                                          │   │
│   │  ┌────────────────────────────────────────────────────┐ │   │
│   │  │              BandwidthStats                         │ │   │
│   │  │  TotalUploaded | TotalDownloaded | CurrentRates    │ │   │
│   │  └────────────────────────────────────────────────────┘ │   │
│   └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## Token Bucket Algorithm

```
┌─────────────────────────────────────────┐
│         TOKEN BUCKET                     │
├─────────────────────────────────────────┤
│                                          │
│    Tokens refill at: bytesPerSecond     │
│                 │                        │
│                 ▼                        │
│    ┌─────────────────────┐              │
│    │ ████████░░░░░░░░░░░ │ ◄─ bucket    │
│    │     (tokens)        │   (max=burst)│
│    └─────────────────────┘              │
│                 │                        │
│                 ▼                        │
│    Request N bytes:                      │
│    - If bucket >= N: consume, proceed   │
│    - Else: wait until enough tokens     │
│                                          │
└─────────────────────────────────────────┘
```

## API

### Basic Limiter

```go
// Create limiter: 1MB/s, 2MB burst
limiter := throttle.NewLimiter(throttle.Limit1MB, throttle.Limit1MB*2)

// Wait for n bytes
ctx := context.Background()
err := limiter.Wait(ctx, 1024) // Wait for 1KB

// Change rate
limiter.SetRate(throttle.Limit5MB)
```

### Throttled Reader/Writer

```go
// Wrap a reader with throttling
file, _ := os.Open("largefile.bin")
throttledReader := throttle.NewThrottledReader(ctx, file, limiter)

// Read at limited rate
buffer := make([]byte, 4096)
n, err := throttledReader.Read(buffer)

// Wrap a writer with throttling
conn, _ := net.Dial("tcp", "peer:6881")
throttledWriter := throttle.NewThrottledWriter(ctx, conn, limiter)
n, err = throttledWriter.Write(data)
```

### Bandwidth Manager

```go
// Create manager: 10MB/s upload, 50MB/s download
manager := throttle.NewBandwidthManager(throttle.Limit10MB, throttle.Limit50MB)

// Wrap connections
reader := manager.WrapReader(ctx, conn)
writer := manager.WrapWriter(ctx, conn)

// Update limits dynamically
manager.SetUploadLimit(throttle.Limit5MB)
manager.SetDownloadLimit(throttle.Limit100MB)

// Get current limits
up, down := manager.GetLimits()

// Get stats
stats := manager.GetStats()
fmt.Printf("Uploaded: %d bytes\n", stats.TotalUploaded)
fmt.Printf("Current rate: %d bytes/s\n", stats.CurrentUpRate)
```

## Preset Limits

| Constant | Value |
|----------|-------|
| `Limit100KB` | 100 KB/s |
| `Limit500KB` | 500 KB/s |
| `Limit1MB` | 1 MB/s |
| `Limit5MB` | 5 MB/s |
| `Limit10MB` | 10 MB/s |
| `Limit50MB` | 50 MB/s |
| `Limit100MB` | 100 MB/s |
| `Unlimited` | No limit |

## Ví dụ: P2P Peer với Throttling

```go
// Khởi tạo bandwidth manager
bwManager := throttle.NewBandwidthManager(
    throttle.Limit10MB,  // Upload limit
    throttle.Limit50MB,  // Download limit
)

// Khi download chunk
func downloadChunk(conn net.Conn, chunkSize int) ([]byte, error) {
    reader := bwManager.WrapReader(ctx, conn)
    buffer := make([]byte, chunkSize)
    _, err := io.ReadFull(reader, buffer)
    return buffer, err
}

// Khi upload chunk
func uploadChunk(conn net.Conn, data []byte) error {
    writer := bwManager.WrapWriter(ctx, conn)
    _, err := writer.Write(data)
    return err
}

// Monitor stats
go func() {
    ticker := time.NewTicker(5 * time.Second)
    for range ticker.C {
        bwManager.UpdateStats()
        stats := bwManager.GetStats()
        log.Printf("Up: %d B/s, Down: %d B/s", 
            stats.CurrentUpRate, stats.CurrentDownRate)
    }
}()
```

## Lưu ý

1. **Burst size**: Cho phép burst ngắn vượt limit để cải thiện latency
2. **Context cancellation**: Hỗ trợ cancel khi waiting
3. **Thread-safe**: Tất cả operations đều thread-safe
4. **Zero allocation**: Không allocate memory trong hot path
5. **Granularity**: Limit áp dụng per-read/write, không phải global

