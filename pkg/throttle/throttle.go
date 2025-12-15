// Package throttle provides bandwidth limiting for network transfers
package throttle

import (
	"context"
	"io"
	"sync"
	"time"
)

// Limiter controls the rate of data transfer
type Limiter struct {
	bytesPerSecond int64
	bucket         int64 // Current available tokens
	maxBucket      int64 // Max tokens (burst size)
	lastUpdate     time.Time
	mu             sync.Mutex
}

// NewLimiter creates a new rate limiter
// bytesPerSecond: maximum bytes per second
// burstSize: maximum burst size (0 = same as bytesPerSecond)
func NewLimiter(bytesPerSecond int64, burstSize int64) *Limiter {
	if burstSize <= 0 {
		burstSize = bytesPerSecond
	}
	return &Limiter{
		bytesPerSecond: bytesPerSecond,
		bucket:         burstSize,
		maxBucket:      burstSize,
		lastUpdate:     time.Now(),
	}
}

// SetRate updates the rate limit
func (l *Limiter) SetRate(bytesPerSecond int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.bytesPerSecond = bytesPerSecond
}

// GetRate returns the current rate limit
func (l *Limiter) GetRate() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.bytesPerSecond
}

// Wait blocks until n bytes can be consumed
func (l *Limiter) Wait(ctx context.Context, n int64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.bytesPerSecond <= 0 {
		return nil // No limit
	}

	// Refill bucket based on elapsed time
	now := time.Now()
	elapsed := now.Sub(l.lastUpdate)
	l.lastUpdate = now

	refill := int64(elapsed.Seconds() * float64(l.bytesPerSecond))
	l.bucket += refill
	if l.bucket > l.maxBucket {
		l.bucket = l.maxBucket
	}

	// If we have enough tokens, consume and return
	if l.bucket >= n {
		l.bucket -= n
		return nil
	}

	// Calculate wait time
	needed := n - l.bucket
	waitTime := time.Duration(float64(needed) / float64(l.bytesPerSecond) * float64(time.Second))
	l.bucket = 0

	l.mu.Unlock()
	select {
	case <-ctx.Done():
		l.mu.Lock()
		return ctx.Err()
	case <-time.After(waitTime):
		l.mu.Lock()
		return nil
	}
}

// ThrottledReader wraps an io.Reader with rate limiting
type ThrottledReader struct {
	reader  io.Reader
	limiter *Limiter
	ctx     context.Context
}

// NewThrottledReader creates a new throttled reader
func NewThrottledReader(ctx context.Context, r io.Reader, limiter *Limiter) *ThrottledReader {
	return &ThrottledReader{
		reader:  r,
		limiter: limiter,
		ctx:     ctx,
	}
}

// Read implements io.Reader with rate limiting
func (r *ThrottledReader) Read(p []byte) (int, error) {
	if err := r.limiter.Wait(r.ctx, int64(len(p))); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

// ThrottledWriter wraps an io.Writer with rate limiting
type ThrottledWriter struct {
	writer  io.Writer
	limiter *Limiter
	ctx     context.Context
}

// NewThrottledWriter creates a new throttled writer
func NewThrottledWriter(ctx context.Context, w io.Writer, limiter *Limiter) *ThrottledWriter {
	return &ThrottledWriter{
		writer:  w,
		limiter: limiter,
		ctx:     ctx,
	}
}

// Write implements io.Writer with rate limiting
func (w *ThrottledWriter) Write(p []byte) (int, error) {
	if err := w.limiter.Wait(w.ctx, int64(len(p))); err != nil {
		return 0, err
	}
	return w.writer.Write(p)
}

// BandwidthManager manages upload/download limits
type BandwidthManager struct {
	uploadLimiter   *Limiter
	downloadLimiter *Limiter
	stats           BandwidthStats
	mu              sync.RWMutex
}

// BandwidthStats tracks bandwidth usage
type BandwidthStats struct {
	TotalUploaded   int64
	TotalDownloaded int64
	CurrentUpRate   int64
	CurrentDownRate int64
	LastUpdateTime  time.Time
	lastUpBytes     int64
	lastDownBytes   int64
}

// NewBandwidthManager creates a new bandwidth manager
// uploadLimit/downloadLimit in bytes per second (0 = unlimited)
func NewBandwidthManager(uploadLimit, downloadLimit int64) *BandwidthManager {
	return &BandwidthManager{
		uploadLimiter:   NewLimiter(uploadLimit, uploadLimit*2),
		downloadLimiter: NewLimiter(downloadLimit, downloadLimit*2),
		stats: BandwidthStats{
			LastUpdateTime: time.Now(),
		},
	}
}

// SetUploadLimit sets the upload rate limit
func (m *BandwidthManager) SetUploadLimit(bytesPerSecond int64) {
	m.uploadLimiter.SetRate(bytesPerSecond)
}

// SetDownloadLimit sets the download rate limit
func (m *BandwidthManager) SetDownloadLimit(bytesPerSecond int64) {
	m.downloadLimiter.SetRate(bytesPerSecond)
}

// GetLimits returns current limits
func (m *BandwidthManager) GetLimits() (upload, download int64) {
	return m.uploadLimiter.GetRate(), m.downloadLimiter.GetRate()
}

// WrapReader wraps a reader with download rate limiting
func (m *BandwidthManager) WrapReader(ctx context.Context, r io.Reader) io.Reader {
	return &trackingReader{
		reader:  NewThrottledReader(ctx, r, m.downloadLimiter),
		manager: m,
	}
}

// WrapWriter wraps a writer with upload rate limiting
func (m *BandwidthManager) WrapWriter(ctx context.Context, w io.Writer) io.Writer {
	return &trackingWriter{
		writer:  NewThrottledWriter(ctx, w, m.uploadLimiter),
		manager: m,
	}
}

// GetStats returns current bandwidth statistics
func (m *BandwidthManager) GetStats() BandwidthStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

// UpdateStats updates current rate calculations
func (m *BandwidthManager) UpdateStats() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(m.stats.LastUpdateTime).Seconds()
	if elapsed > 0 {
		upDelta := m.stats.TotalUploaded - m.stats.lastUpBytes
		downDelta := m.stats.TotalDownloaded - m.stats.lastDownBytes

		m.stats.CurrentUpRate = int64(float64(upDelta) / elapsed)
		m.stats.CurrentDownRate = int64(float64(downDelta) / elapsed)

		m.stats.lastUpBytes = m.stats.TotalUploaded
		m.stats.lastDownBytes = m.stats.TotalDownloaded
		m.stats.LastUpdateTime = now
	}
}

// trackingReader tracks bytes read
type trackingReader struct {
	reader  io.Reader
	manager *BandwidthManager
}

func (r *trackingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.manager.mu.Lock()
		r.manager.stats.TotalDownloaded += int64(n)
		r.manager.mu.Unlock()
	}
	return n, err
}

// trackingWriter tracks bytes written
type trackingWriter struct {
	writer  io.Writer
	manager *BandwidthManager
}

func (w *trackingWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if n > 0 {
		w.manager.mu.Lock()
		w.manager.stats.TotalUploaded += int64(n)
		w.manager.mu.Unlock()
	}
	return n, err
}

// Common rate constants
const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB

	// Common limits
	Limit100KB = 100 * KB
	Limit500KB = 500 * KB
	Limit1MB   = 1 * MB
	Limit5MB   = 5 * MB
	Limit10MB  = 10 * MB
	Limit50MB  = 50 * MB
	Limit100MB = 100 * MB
	Unlimited  = 0
)
