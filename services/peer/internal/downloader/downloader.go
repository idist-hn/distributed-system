package downloader

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/p2p-filesharing/distributed-system/pkg/chunker"
	"github.com/p2p-filesharing/distributed-system/pkg/hash"
	"github.com/p2p-filesharing/distributed-system/pkg/protocol"
	"github.com/p2p-filesharing/distributed-system/services/peer/internal/p2p"
	"github.com/p2p-filesharing/distributed-system/services/peer/internal/storage"
)

// DownloadStats tracks download statistics
type DownloadStats struct {
	TotalChunks      int
	DownloadedChunks int32
	FailedChunks     int32
	BytesDownloaded  int64
	StartTime        time.Time
	EndTime          time.Time
	ActiveWorkers    int32
	PeerStats        map[string]*PeerDownloadStats
	mu               sync.RWMutex
}

// PeerDownloadStats tracks per-peer statistics
type PeerDownloadStats struct {
	PeerID           string
	ChunksDownloaded int32
	BytesDownloaded  int64
	Failures         int32
	AvgLatency       time.Duration
	LastLatency      time.Duration
	Score            float64 // Higher is better
}

// ChunkTask represents a chunk download task
type ChunkTask struct {
	Index         int
	Hash          string
	Size          int64
	Retries       int
	MaxRetries    int
	PreferredPeer string // Optional preferred peer
}

// Downloader handles file downloads from peers with parallel chunk support
type Downloader struct {
	storage      *storage.LocalStorage
	p2pClient    *p2p.Client
	chunker      *chunker.Chunker
	maxWorkers   int
	chunkTimeout time.Duration
	maxRetries   int
}

// New creates a new Downloader
func New(store *storage.LocalStorage, client *p2p.Client) *Downloader {
	return &Downloader{
		storage:      store,
		p2pClient:    client,
		chunker:      chunker.New(chunker.DefaultChunkSize),
		maxWorkers:   8,                // Download from up to 8 peers concurrently
		chunkTimeout: 30 * time.Second, // Timeout per chunk
		maxRetries:   3,                // Max retries per chunk
	}
}

// NewWithConfig creates a Downloader with custom configuration
func NewWithConfig(store *storage.LocalStorage, client *p2p.Client, maxWorkers, maxRetries int, chunkTimeout time.Duration) *Downloader {
	return &Downloader{
		storage:      store,
		p2pClient:    client,
		chunker:      chunker.New(chunker.DefaultChunkSize),
		maxWorkers:   maxWorkers,
		chunkTimeout: chunkTimeout,
		maxRetries:   maxRetries,
	}
}

// DownloadFile downloads a file from available peers using parallel chunk downloads
func (d *Downloader) DownloadFile(fileInfo *protocol.GetPeersResponse) error {
	if len(fileInfo.Peers) == 0 {
		return fmt.Errorf("no peers available for this file")
	}

	// Initialize download state
	metadata := &protocol.FileMetadata{
		Name:      fileInfo.FileName,
		Size:      fileInfo.FileSize,
		Hash:      fileInfo.FileHash,
		ChunkSize: fileInfo.ChunkSize,
		Chunks:    fileInfo.Chunks,
	}

	state := d.storage.StartDownload(metadata)
	stats := d.initStats(len(metadata.Chunks), fileInfo.Peers)

	log.Printf("[Downloader] Starting parallel download: %s (%d chunks from %d peers)",
		metadata.Name, len(metadata.Chunks), len(fileInfo.Peers))

	// Create chunk task queue with prioritization
	taskQueue := make(chan *ChunkTask, len(metadata.Chunks))
	retryQueue := make(chan *ChunkTask, len(metadata.Chunks))

	// Initialize tasks
	for i, chunk := range metadata.Chunks {
		if !state.ChunksReceived[i] {
			taskQueue <- &ChunkTask{
				Index:      i,
				Hash:       chunk.Hash,
				Size:       chunk.Size,
				MaxRetries: d.maxRetries,
			}
		}
	}
	close(taskQueue)

	// Determine optimal worker count
	numWorkers := min(d.maxWorkers, len(fileInfo.Peers), len(metadata.Chunks))
	log.Printf("[Downloader] Using %d parallel workers", numWorkers)

	// Create worker pool with peer assignment
	var wg sync.WaitGroup
	results := make(chan *chunkResult, len(metadata.Chunks))
	workerDone := make(chan struct{})

	// Start workers - each gets assigned peers in round-robin
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		assignedPeers := d.assignPeers(i, numWorkers, fileInfo.Peers)
		go d.parallelWorker(&wg, i, assignedPeers, metadata, state, stats, taskQueue, retryQueue, results)
	}

	// Start retry processor
	go d.processRetries(retryQueue, taskQueue, workerDone)

	// Wait for workers and collect results
	go func() {
		wg.Wait()
		close(workerDone)
		close(results)
	}()

	// Process results
	var lastErr error
	for result := range results {
		if result.err != nil {
			lastErr = result.err
		}
	}

	// Close retry queue
	close(retryQueue)

	// Calculate stats
	stats.EndTime = time.Now()
	d.logDownloadStats(stats, metadata.Name)

	// Verify download complete
	if !d.storage.IsDownloadComplete(metadata.Hash) {
		if lastErr != nil {
			return fmt.Errorf("download incomplete: %w", lastErr)
		}
		return fmt.Errorf("download incomplete")
	}

	// Assemble file
	if err := d.assembleFile(state); err != nil {
		return fmt.Errorf("failed to assemble file: %w", err)
	}

	// Move to shared files
	d.storage.AddSharedFile(metadata, state.OutputPath)

	log.Printf("[Downloader] Download complete: %s (%.2f MB/s)",
		metadata.Name, d.calculateSpeed(stats))
	return nil
}

// chunkResult represents the result of downloading a chunk
type chunkResult struct {
	index int
	size  int64
	err   error
}

// initStats initializes download statistics
func (d *Downloader) initStats(totalChunks int, peers []protocol.PeerFileInfo) *DownloadStats {
	stats := &DownloadStats{
		TotalChunks: totalChunks,
		StartTime:   time.Now(),
		PeerStats:   make(map[string]*PeerDownloadStats),
	}
	for _, peer := range peers {
		stats.PeerStats[peer.PeerID] = &PeerDownloadStats{
			PeerID: peer.PeerID,
			Score:  100.0, // Initial score
		}
	}
	return stats
}

// assignPeers assigns peers to a worker in round-robin fashion
func (d *Downloader) assignPeers(workerID, numWorkers int, peers []protocol.PeerFileInfo) []protocol.PeerFileInfo {
	var assigned []protocol.PeerFileInfo
	for i, peer := range peers {
		if i%numWorkers == workerID {
			assigned = append(assigned, peer)
		}
	}
	// If no peers assigned, give at least the first available
	if len(assigned) == 0 && len(peers) > 0 {
		assigned = append(assigned, peers[workerID%len(peers)])
	}
	return assigned
}

// downloadWorker downloads chunks assigned to it
func (d *Downloader) downloadWorker(wg *sync.WaitGroup, workerID int, peer protocol.PeerFileInfo, metadata *protocol.FileMetadata, state *storage.DownloadState, chunks <-chan int, errors chan<- error) {
	defer wg.Done()

	// Connect to peer
	conn, err := d.p2pClient.Connect(peer.IP, peer.Port)
	if err != nil {
		log.Printf("[Worker %d] Failed to connect to peer %s: %v", workerID, peer.PeerID, err)
		// Put chunks back or let other workers handle them
		return
	}
	defer conn.Close()

	for chunkIndex := range chunks {
		// Skip if already downloaded
		if state.ChunksReceived[chunkIndex] {
			continue
		}

		expectedHash := metadata.Chunks[chunkIndex].Hash

		// Request chunk
		data, err := conn.RequestChunk(metadata.Hash, chunkIndex, expectedHash)
		if err != nil {
			log.Printf("[Worker %d] Failed to get chunk %d: %v", workerID, chunkIndex, err)
			errors <- err
			continue
		}

		// Verify hash
		if !hash.Verify(data, expectedHash) {
			errors <- fmt.Errorf("chunk %d hash mismatch", chunkIndex)
			continue
		}

		// Save chunk
		chunkPath := filepath.Join(state.TempDir, fmt.Sprintf("chunk_%d", chunkIndex))
		if err := os.WriteFile(chunkPath, data, 0644); err != nil {
			errors <- err
			continue
		}

		d.storage.MarkChunkReceived(metadata.Hash, chunkIndex)
		log.Printf("[Worker %d] Downloaded chunk %d/%d", workerID, chunkIndex+1, len(metadata.Chunks))
	}
}

// assembleFile combines all chunks into the final file
func (d *Downloader) assembleFile(state *storage.DownloadState) error {
	outFile, err := os.Create(state.OutputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	for i := range state.ChunksReceived {
		chunkPath := filepath.Join(state.TempDir, fmt.Sprintf("chunk_%d", i))
		data, err := os.ReadFile(chunkPath)
		if err != nil {
			return err
		}

		if _, err := outFile.Write(data); err != nil {
			return err
		}
	}

	// Cleanup temp files
	os.RemoveAll(state.TempDir)

	return nil
}

// parallelWorker downloads chunks from multiple peers in parallel
func (d *Downloader) parallelWorker(
	wg *sync.WaitGroup,
	workerID int,
	peers []protocol.PeerFileInfo,
	metadata *protocol.FileMetadata,
	state *storage.DownloadState,
	stats *DownloadStats,
	tasks <-chan *ChunkTask,
	retryQueue chan<- *ChunkTask,
	results chan<- *chunkResult,
) {
	defer wg.Done()

	// Sort peers by score (best first)
	sortedPeers := d.sortPeersByScore(peers, stats)

	// Track active peer connection
	var currentConn *p2p.PeerConnection
	var currentPeerIdx int

	// Process tasks
	for task := range tasks {
		// Skip if already downloaded
		if state.ChunksReceived[task.Index] {
			continue
		}

		var data []byte
		var err error
		var downloadedFromPeer string
		startTime := time.Now()

		// Try each peer until success
		for attempt := 0; attempt < len(sortedPeers); attempt++ {
			peerIdx := (currentPeerIdx + attempt) % len(sortedPeers)
			peer := sortedPeers[peerIdx]

			// Connect if needed
			if currentConn == nil || peerIdx != currentPeerIdx {
				if currentConn != nil {
					currentConn.Close()
				}
				currentConn, err = d.p2pClient.Connect(peer.IP, peer.Port)
				if err != nil {
					d.updatePeerScore(stats, peer.PeerID, false, 0)
					continue
				}
				currentPeerIdx = peerIdx
			}

			// Request chunk
			data, err = currentConn.RequestChunk(metadata.Hash, task.Index, task.Hash)
			if err == nil {
				// Verify hash
				if hash.Verify(data, task.Hash) {
					downloadedFromPeer = peer.PeerID
					break
				}
				err = fmt.Errorf("hash mismatch")
			}

			// Update peer score on failure
			d.updatePeerScore(stats, peer.PeerID, false, 0)
			currentConn.Close()
			currentConn = nil
		}

		latency := time.Since(startTime)

		if err != nil || data == nil {
			// Retry if possible
			task.Retries++
			if task.Retries < task.MaxRetries {
				select {
				case retryQueue <- task:
				default:
					// Queue full, log error
					log.Printf("[Worker %d] Retry queue full for chunk %d", workerID, task.Index)
				}
			}
			results <- &chunkResult{index: task.Index, err: err}
			continue
		}

		// Save chunk
		chunkPath := filepath.Join(state.TempDir, fmt.Sprintf("chunk_%d", task.Index))
		if err := os.WriteFile(chunkPath, data, 0644); err != nil {
			results <- &chunkResult{index: task.Index, err: err}
			continue
		}

		// Update stats and state
		d.storage.MarkChunkReceived(metadata.Hash, task.Index)
		d.updatePeerScore(stats, downloadedFromPeer, true, latency)

		stats.mu.Lock()
		stats.DownloadedChunks++
		stats.BytesDownloaded += int64(len(data))
		stats.mu.Unlock()

		progress := float64(stats.DownloadedChunks) / float64(stats.TotalChunks) * 100
		log.Printf("[Worker %d] Chunk %d/%d (%.1f%%) from %s in %v",
			workerID, task.Index+1, stats.TotalChunks, progress,
			downloadedFromPeer[:8], latency)

		results <- &chunkResult{index: task.Index, size: int64(len(data))}
	}

	if currentConn != nil {
		currentConn.Close()
	}
}

// sortPeersByScore sorts peers by their download score
func (d *Downloader) sortPeersByScore(peers []protocol.PeerFileInfo, stats *DownloadStats) []protocol.PeerFileInfo {
	sorted := make([]protocol.PeerFileInfo, len(peers))
	copy(sorted, peers)

	stats.mu.RLock()
	defer stats.mu.RUnlock()

	sort.Slice(sorted, func(i, j int) bool {
		scoreI := stats.PeerStats[sorted[i].PeerID].Score
		scoreJ := stats.PeerStats[sorted[j].PeerID].Score
		return scoreI > scoreJ
	})

	return sorted
}

// updatePeerScore updates peer's score based on performance
func (d *Downloader) updatePeerScore(stats *DownloadStats, peerID string, success bool, latency time.Duration) {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	peerStats, exists := stats.PeerStats[peerID]
	if !exists {
		return
	}

	if success {
		peerStats.ChunksDownloaded++
		peerStats.LastLatency = latency

		// Update average latency
		if peerStats.AvgLatency == 0 {
			peerStats.AvgLatency = latency
		} else {
			peerStats.AvgLatency = (peerStats.AvgLatency + latency) / 2
		}

		// Increase score for success, bonus for fast response
		speedBonus := 10.0
		if latency < 100*time.Millisecond {
			speedBonus = 20.0
		} else if latency > time.Second {
			speedBonus = 5.0
		}
		peerStats.Score = min(200.0, peerStats.Score+speedBonus)
	} else {
		peerStats.Failures++
		// Decrease score for failure
		peerStats.Score = max(0.0, peerStats.Score-25.0)
	}
}

// processRetries handles retry queue - moves tasks back to main queue
func (d *Downloader) processRetries(retryQueue <-chan *ChunkTask, taskQueue chan<- *ChunkTask, done <-chan struct{}) {
	for {
		select {
		case task, ok := <-retryQueue:
			if !ok {
				return
			}
			// Delay before retry
			time.Sleep(time.Duration(task.Retries) * 500 * time.Millisecond)
			select {
			case taskQueue <- task:
			case <-done:
				return
			}
		case <-done:
			return
		}
	}
}

// calculateSpeed calculates download speed in MB/s
func (d *Downloader) calculateSpeed(stats *DownloadStats) float64 {
	duration := stats.EndTime.Sub(stats.StartTime).Seconds()
	if duration == 0 {
		return 0
	}
	return float64(stats.BytesDownloaded) / 1024 / 1024 / duration
}

// logDownloadStats logs download statistics
func (d *Downloader) logDownloadStats(stats *DownloadStats, fileName string) {
	duration := stats.EndTime.Sub(stats.StartTime)
	speed := d.calculateSpeed(stats)

	log.Printf("[Stats] File: %s", fileName)
	log.Printf("[Stats] Total chunks: %d, Downloaded: %d, Failed: %d",
		stats.TotalChunks, stats.DownloadedChunks, stats.FailedChunks)
	log.Printf("[Stats] Bytes: %d, Duration: %v, Speed: %.2f MB/s",
		stats.BytesDownloaded, duration, speed)

	stats.mu.RLock()
	defer stats.mu.RUnlock()

	log.Printf("[Stats] Peer performance:")
	for peerID, peerStats := range stats.PeerStats {
		if peerStats.ChunksDownloaded > 0 {
			log.Printf("  - %s: %d chunks, avg latency %v, score %.1f",
				peerID[:8], peerStats.ChunksDownloaded, peerStats.AvgLatency, peerStats.Score)
		}
	}
}

// GetDownloadStats returns current download statistics (for monitoring)
func (d *Downloader) GetDownloadStats(fileHash string) (*DownloadStats, bool) {
	// This could be extended to track active downloads
	return nil, false
}
