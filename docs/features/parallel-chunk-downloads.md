# Parallel Chunk Downloads

## Tổng quan

Tính năng **Parallel Chunk Downloads** cho phép tải nhiều chunks của file cùng lúc từ nhiều peers khác nhau, giúp tăng tốc độ download đáng kể.

## Kiến trúc

```
┌─────────────────────────────────────────────────────────────────┐
│                    PARALLEL DOWNLOAD ENGINE                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌─────────────┐     ┌─────────────┐     ┌─────────────┐       │
│   │  Worker 1   │     │  Worker 2   │     │  Worker N   │       │
│   │  (Peer A)   │     │  (Peer B)   │     │  (Peer X)   │       │
│   └──────┬──────┘     └──────┬──────┘     └──────┬──────┘       │
│          │                   │                   │               │
│          ▼                   ▼                   ▼               │
│   ┌─────────────────────────────────────────────────────┐       │
│   │                   Task Queue                         │       │
│   │   [Chunk 1] [Chunk 2] [Chunk 3] ... [Chunk N]       │       │
│   └─────────────────────────────────────────────────────┘       │
│                              │                                   │
│                              ▼                                   │
│   ┌─────────────────────────────────────────────────────┐       │
│   │                  Retry Queue                         │       │
│   │   Failed chunks with exponential backoff            │       │
│   └─────────────────────────────────────────────────────┘       │
│                              │                                   │
│                              ▼                                   │
│   ┌─────────────────────────────────────────────────────┐       │
│   │                 File Assembler                       │       │
│   │   Combines chunks → Final file                      │       │
│   └─────────────────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────────────────┘
```

## Tính năng chính

### 1. Multi-worker Architecture
- Mặc định 8 workers chạy song song
- Mỗi worker được gán một tập peers theo round-robin
- Workers độc lập, không block lẫn nhau

### 2. Smart Peer Selection
- **Peer Scoring**: Mỗi peer có điểm số dựa trên hiệu suất
- **Dynamic Ranking**: Peers được sắp xếp theo score, ưu tiên peer nhanh nhất
- **Automatic Failover**: Tự động chuyển sang peer khác khi gặp lỗi

### 3. Retry Mechanism
- Mỗi chunk có tối đa 3 lần retry
- Exponential backoff: delay tăng theo số lần retry
- Failed chunks được đưa vào retry queue

### 4. Statistics & Monitoring
- Theo dõi số chunks đã tải, tốc độ, latency
- Per-peer statistics (chunks, latency, failures)
- Tính toán download speed realtime

## API

### Downloader

```go
// Tạo downloader với cấu hình mặc định
dl := downloader.New(storage, p2pClient)

// Tạo downloader với cấu hình tùy chỉnh
dl := downloader.NewWithConfig(
    storage, 
    p2pClient,
    maxWorkers,     // Số workers (default: 8)
    maxRetries,     // Số lần retry (default: 3)  
    chunkTimeout,   // Timeout mỗi chunk (default: 30s)
)

// Download file
err := dl.DownloadFile(fileInfo)
```

### Download Statistics

```go
type DownloadStats struct {
    TotalChunks      int
    DownloadedChunks int32
    FailedChunks     int32
    BytesDownloaded  int64
    StartTime        time.Time
    EndTime          time.Time
    PeerStats        map[string]*PeerDownloadStats
}

type PeerDownloadStats struct {
    PeerID           string
    ChunksDownloaded int32
    BytesDownloaded  int64
    Failures         int32
    AvgLatency       time.Duration
    Score            float64
}
```

## Peer Scoring Algorithm

```
Score = 100 (initial)

On Success:
  - latency < 100ms: Score += 20
  - latency < 1s:    Score += 10
  - latency >= 1s:   Score += 5
  - Max score: 200

On Failure:
  - Score -= 25
  - Min score: 0
```

## Cấu hình

| Parameter | Default | Description |
|-----------|---------|-------------|
| `maxWorkers` | 8 | Số worker chạy song song |
| `chunkTimeout` | 30s | Timeout cho mỗi chunk |
| `maxRetries` | 3 | Số lần retry mỗi chunk |

## Ví dụ Output

```
[Downloader] Starting parallel download: video.mp4 (100 chunks from 5 peers)
[Downloader] Using 5 parallel workers
[Worker 0] Chunk 1/100 (1.0%) from 6d9b0f7c in 45ms
[Worker 1] Chunk 2/100 (2.0%) from 38a9e787 in 52ms
[Worker 2] Chunk 3/100 (3.0%) from 6d9b0f7c in 38ms
...
[Stats] File: video.mp4
[Stats] Total chunks: 100, Downloaded: 100, Failed: 0
[Stats] Bytes: 104857600, Duration: 12.5s, Speed: 8.00 MB/s
[Stats] Peer performance:
  - 6d9b0f7c: 40 chunks, avg latency 42ms, score 180.0
  - 38a9e787: 35 chunks, avg latency 55ms, score 165.0
  - ab12cd34: 25 chunks, avg latency 78ms, score 150.0
[Downloader] Download complete: video.mp4 (8.00 MB/s)
```

## So sánh hiệu suất

| Scenario | Sequential | Parallel (8 workers) | Improvement |
|----------|------------|---------------------|-------------|
| 100MB file, 1 peer | 10s | 10s | - |
| 100MB file, 4 peers | 10s | 2.5s | 4x |
| 100MB file, 8 peers | 10s | 1.25s | 8x |
| 1GB file, 8 peers | 100s | 12.5s | 8x |

## Lưu ý

1. **Network bandwidth**: Tốc độ thực tế phụ thuộc vào bandwidth của peers và network
2. **Peer availability**: Nếu peers ít hơn workers, số workers sẽ tự động giảm
3. **Memory usage**: Mỗi worker giữ 1 connection, không buffer toàn bộ file

