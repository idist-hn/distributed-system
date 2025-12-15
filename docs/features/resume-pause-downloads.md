# Resume/Pause Downloads

## Tổng quan

Tính năng **Resume/Pause Downloads** cho phép tạm dừng và tiếp tục download, lưu trạng thái chunks đã tải vào disk để có thể resume sau khi khởi động lại ứng dụng.

## Kiến trúc

```
┌─────────────────────────────────────────────────────────────────┐
│                    DOWNLOAD STATE MANAGEMENT                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌─────────────┐                    ┌─────────────────────┐    │
│   │   Memory    │◄──── Save ────────►│    state.json       │    │
│   │   State     │                    │    (Persistent)     │    │
│   └──────┬──────┘                    └─────────────────────┘    │
│          │                                                       │
│   ┌──────▼──────────────────────────────────────────────────┐   │
│   │                   Download States                        │   │
│   │                                                          │   │
│   │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐    │   │
│   │  │ Pending │──► Active  │──► Paused  │──► Active  │    │   │
│   │  └─────────┘  └────┬────┘  └─────────┘  └────┬────┘    │   │
│   │                    │                          │          │   │
│   │                    ▼                          ▼          │   │
│   │              ┌───────────┐            ┌───────────┐     │   │
│   │              │ Completed │            │ Completed │     │   │
│   │              └───────────┘            └───────────┘     │   │
│   │                                                          │   │
│   │  ┌─────────┐                         ┌───────────┐      │   │
│   │  │ Failed  │◄──── Retry ────────────►│  Active   │      │   │
│   │  └─────────┘                         └───────────┘      │   │
│   └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## Download Status Flow

```
pending ──► active ──► completed
              │
              ├──► paused ──► active (resume)
              │
              └──► failed ──► active (retry)
                     │
                     └──► cancelled (cleanup)
```

## Tính năng chính

### 1. Pause Download
- Đánh dấu download là `paused`
- Lưu trạng thái vào disk ngay lập tức
- Workers dừng ngay sau chunk hiện tại

### 2. Resume Download
- Load trạng thái từ disk
- Tiếp tục từ chunk cuối cùng đã tải
- Không cần tải lại chunks đã hoàn thành

### 3. Auto-Resume on Restart
- Khi khởi động, load state.json
- Downloads đang active được đánh dấu paused
- User có thể resume thủ công

### 4. Cancel Download
- Xóa chunks tạm thời
- Xóa download khỏi danh sách
- Giải phóng dung lượng đĩa

## API

### Storage Methods

```go
// Pause download
err := storage.PauseDownload(fileHash)

// Resume download
state, err := storage.ResumeDownload(fileHash)

// Cancel download
err := storage.CancelDownload(fileHash)

// Get progress
progress, err := storage.GetDownloadProgress(fileHash) // Returns 0-100

// List all downloads
downloads := storage.ListDownloads()

// Get paused downloads
paused := storage.GetPausedDownloads()
```

### DownloadState Structure

```go
type DownloadState struct {
    Metadata        *protocol.FileMetadata
    ChunksReceived  []bool           // Tracks which chunks are done
    TempDir         string           // Temp storage for chunks
    OutputPath      string           // Final file path
    Status          DownloadStatus   // pending/active/paused/completed/failed
    StartedAt       time.Time
    PausedAt        *time.Time       // When paused
    CompletedAt     *time.Time       // When completed
    BytesDownloaded int64
    TotalBytes      int64
    LastError       string           // Last error message
    RetryCount      int              // Number of retries
}
```

### Download Status

| Status | Description |
|--------|-------------|
| `pending` | Đã khởi tạo, chưa bắt đầu |
| `active` | Đang download |
| `paused` | Đã tạm dừng bởi user |
| `completed` | Hoàn thành thành công |
| `failed` | Thất bại, có thể retry |
| `cancelled` | Đã hủy, cleanup xong |

## State Persistence

### File Format (state.json)

```json
{
  "shared_files": {
    "abc123...": {
      "metadata": {...},
      "file_path": "/path/to/file"
    }
  },
  "downloads": {
    "def456...": {
      "metadata": {...},
      "chunks_received": [true, true, false, false, ...],
      "temp_dir": "/path/to/temp/def456",
      "output_path": "/path/to/downloads/file.mp4",
      "status": "paused",
      "started_at": "2024-01-15T10:00:00Z",
      "paused_at": "2024-01-15T10:05:00Z",
      "bytes_downloaded": 52428800,
      "total_bytes": 104857600
    }
  }
}
```

## Ví dụ sử dụng

```go
// Start download
dl := downloader.New(storage, p2pClient)
go dl.DownloadFile(fileInfo)

// Pause after 5 seconds
time.Sleep(5 * time.Second)
storage.PauseDownload(fileHash)

// Resume later
state, _ := storage.ResumeDownload(fileHash)
dl.DownloadFile(fileInfo) // Continues from where it left off

// Check progress
progress, _ := storage.GetDownloadProgress(fileHash)
fmt.Printf("Progress: %.1f%%\n", progress)
```

## Lưu ý

1. **State is saved immediately** khi pause để tránh mất dữ liệu
2. **Chunks are stored in temp dir** cho đến khi download hoàn tất
3. **On restart**, active downloads tự động chuyển sang paused
4. **Retry mechanism** tích hợp với parallel download

