package throttle

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

func TestNewLimiter(t *testing.T) {
	tests := []struct {
		name           string
		bytesPerSecond int64
		burstSize      int64
		wantBurst      int64
	}{
		{"normal", 1000, 2000, 2000},
		{"zero burst", 1000, 0, 1000},
		{"negative burst", 1000, -1, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLimiter(tt.bytesPerSecond, tt.burstSize)
			if l.bytesPerSecond != tt.bytesPerSecond {
				t.Errorf("bytesPerSecond = %d, want %d", l.bytesPerSecond, tt.bytesPerSecond)
			}
			if l.maxBucket != tt.wantBurst {
				t.Errorf("maxBucket = %d, want %d", l.maxBucket, tt.wantBurst)
			}
		})
	}
}

func TestLimiter_SetGetRate(t *testing.T) {
	l := NewLimiter(1000, 0)
	if got := l.GetRate(); got != 1000 {
		t.Errorf("GetRate() = %d, want 1000", got)
	}

	l.SetRate(2000)
	if got := l.GetRate(); got != 2000 {
		t.Errorf("GetRate() after SetRate = %d, want 2000", got)
	}
}

func TestLimiter_Wait_NoLimit(t *testing.T) {
	l := NewLimiter(0, 0) // No limit
	ctx := context.Background()

	start := time.Now()
	err := l.Wait(ctx, 1000000)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Wait() error = %v", err)
	}
	if elapsed > 10*time.Millisecond {
		t.Errorf("Wait() took %v, expected instant", elapsed)
	}
}

func TestLimiter_Wait_WithBurst(t *testing.T) {
	l := NewLimiter(1000, 1000)
	ctx := context.Background()

	// First call should be instant (using burst)
	start := time.Now()
	err := l.Wait(ctx, 500)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Wait() error = %v", err)
	}
	if elapsed > 10*time.Millisecond {
		t.Errorf("First Wait() took %v, expected instant", elapsed)
	}
}

func TestLimiter_Wait_ContextCanceled(t *testing.T) {
	l := NewLimiter(100, 0) // Very slow
	l.bucket = 0            // Empty bucket

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := l.Wait(ctx, 10000)
	if err != context.Canceled {
		t.Errorf("Wait() error = %v, want context.Canceled", err)
	}
}

func TestThrottledReader(t *testing.T) {
	data := []byte("hello world")
	reader := bytes.NewReader(data)
	limiter := NewLimiter(Unlimited, 0)
	ctx := context.Background()

	tr := NewThrottledReader(ctx, reader, limiter)
	buf := make([]byte, len(data))
	n, err := tr.Read(buf)

	if err != nil {
		t.Errorf("Read() error = %v", err)
	}
	if n != len(data) {
		t.Errorf("Read() n = %d, want %d", n, len(data))
	}
	if !bytes.Equal(buf, data) {
		t.Errorf("Read() data = %s, want %s", buf, data)
	}
}

func TestThrottledWriter(t *testing.T) {
	var buf bytes.Buffer
	limiter := NewLimiter(Unlimited, 0)
	ctx := context.Background()

	tw := NewThrottledWriter(ctx, &buf, limiter)
	data := []byte("hello world")
	n, err := tw.Write(data)

	if err != nil {
		t.Errorf("Write() error = %v", err)
	}
	if n != len(data) {
		t.Errorf("Write() n = %d, want %d", n, len(data))
	}
	if !bytes.Equal(buf.Bytes(), data) {
		t.Errorf("Write() data = %s, want %s", buf.Bytes(), data)
	}
}

func TestBandwidthManager(t *testing.T) {
	bm := NewBandwidthManager(Limit1MB, Limit1MB)

	up, down := bm.GetLimits()
	if up != Limit1MB || down != Limit1MB {
		t.Errorf("GetLimits() = (%d, %d), want (%d, %d)", up, down, Limit1MB, Limit1MB)
	}

	bm.SetUploadLimit(Limit5MB)
	bm.SetDownloadLimit(Limit10MB)

	up, down = bm.GetLimits()
	if up != Limit5MB || down != Limit10MB {
		t.Errorf("GetLimits() after set = (%d, %d), want (%d, %d)", up, down, Limit5MB, Limit10MB)
	}
}

func TestBandwidthManager_WrapReader(t *testing.T) {
	bm := NewBandwidthManager(Unlimited, Unlimited)
	data := []byte("test data for reading")
	reader := bytes.NewReader(data)
	ctx := context.Background()

	wrapped := bm.WrapReader(ctx, reader)
	buf := make([]byte, len(data))
	n, err := io.ReadFull(wrapped, buf)

	if err != nil {
		t.Errorf("Read() error = %v", err)
	}
	if n != len(data) {
		t.Errorf("Read() n = %d, want %d", n, len(data))
	}

	stats := bm.GetStats()
	if stats.TotalDownloaded != int64(len(data)) {
		t.Errorf("TotalDownloaded = %d, want %d", stats.TotalDownloaded, len(data))
	}
}

func TestBandwidthManager_WrapWriter(t *testing.T) {
	bm := NewBandwidthManager(Unlimited, Unlimited)
	var buf bytes.Buffer
	ctx := context.Background()

	wrapped := bm.WrapWriter(ctx, &buf)
	data := []byte("test data for writing")
	n, err := wrapped.Write(data)

	if err != nil {
		t.Errorf("Write() error = %v", err)
	}
	if n != len(data) {
		t.Errorf("Write() n = %d, want %d", n, len(data))
	}

	stats := bm.GetStats()
	if stats.TotalUploaded != int64(len(data)) {
		t.Errorf("TotalUploaded = %d, want %d", stats.TotalUploaded, len(data))
	}
}

func TestConstants(t *testing.T) {
	if KB != 1024 {
		t.Errorf("KB = %d, want 1024", KB)
	}
	if MB != 1024*1024 {
		t.Errorf("MB = %d, want %d", MB, 1024*1024)
	}
	if GB != 1024*1024*1024 {
		t.Errorf("GB = %d, want %d", GB, 1024*1024*1024)
	}
	if Limit1MB != MB {
		t.Errorf("Limit1MB = %d, want %d", Limit1MB, MB)
	}
}

