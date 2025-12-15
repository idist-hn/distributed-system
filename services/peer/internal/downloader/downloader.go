package downloader

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/p2p-filesharing/distributed-system/pkg/chunker"
	"github.com/p2p-filesharing/distributed-system/pkg/hash"
	"github.com/p2p-filesharing/distributed-system/pkg/protocol"
	"github.com/p2p-filesharing/distributed-system/services/peer/internal/p2p"
	"github.com/p2p-filesharing/distributed-system/services/peer/internal/storage"
)

// Downloader handles file downloads from peers
type Downloader struct {
	storage    *storage.LocalStorage
	p2pClient  *p2p.Client
	chunker    *chunker.Chunker
	maxWorkers int
}

// New creates a new Downloader
func New(store *storage.LocalStorage, client *p2p.Client) *Downloader {
	return &Downloader{
		storage:    store,
		p2pClient:  client,
		chunker:    chunker.New(chunker.DefaultChunkSize),
		maxWorkers: 4, // Download from 4 peers concurrently
	}
}

// DownloadFile downloads a file from available peers
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
	log.Printf("[Downloader] Starting download: %s (%d chunks)", metadata.Name, len(metadata.Chunks))

	// Create chunk queue
	chunkQueue := make(chan int, len(metadata.Chunks))
	for i := range metadata.Chunks {
		chunkQueue <- i
	}
	close(chunkQueue)

	// Create worker pool
	var wg sync.WaitGroup
	errors := make(chan error, len(metadata.Chunks))

	numWorkers := d.maxWorkers
	if len(fileInfo.Peers) < numWorkers {
		numWorkers = len(fileInfo.Peers)
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go d.downloadWorker(&wg, i, fileInfo.Peers[i%len(fileInfo.Peers)], metadata, state, chunkQueue, errors)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		if err != nil {
			return err
		}
	}

	// Verify download complete
	if !d.storage.IsDownloadComplete(metadata.Hash) {
		return fmt.Errorf("download incomplete")
	}

	// Assemble file
	if err := d.assembleFile(state); err != nil {
		return fmt.Errorf("failed to assemble file: %w", err)
	}

	// Move to shared files
	d.storage.AddSharedFile(metadata, state.OutputPath)

	log.Printf("[Downloader] Download complete: %s", metadata.Name)
	return nil
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
