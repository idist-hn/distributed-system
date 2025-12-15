package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/p2p-filesharing/distributed-system/pkg/protocol"
)

// LocalStorage manages local file storage for a peer
type LocalStorage struct {
	mu          sync.RWMutex
	baseDir     string
	sharedFiles map[string]*SharedFile    // fileHash -> SharedFile
	downloads   map[string]*DownloadState // fileHash -> DownloadState
}

// SharedFile represents a file being shared by this peer
type SharedFile struct {
	Metadata *protocol.FileMetadata `json:"metadata"`
	FilePath string                 `json:"file_path"`
}

// DownloadState tracks the progress of a file download
type DownloadState struct {
	Metadata       *protocol.FileMetadata `json:"metadata"`
	ChunksReceived []bool                 `json:"chunks_received"`
	TempDir        string                 `json:"temp_dir"`
	OutputPath     string                 `json:"output_path"`
}

// NewLocalStorage creates a new local storage manager
func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	// Create directories
	dirs := []string{
		filepath.Join(baseDir, "shared"),
		filepath.Join(baseDir, "downloads"),
		filepath.Join(baseDir, "temp"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	return &LocalStorage{
		baseDir:     baseDir,
		sharedFiles: make(map[string]*SharedFile),
		downloads:   make(map[string]*DownloadState),
	}, nil
}

// AddSharedFile adds a file to the shared files list
func (s *LocalStorage) AddSharedFile(metadata *protocol.FileMetadata, filePath string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sharedFiles[metadata.Hash] = &SharedFile{
		Metadata: metadata,
		FilePath: filePath,
	}
}

// GetSharedFile retrieves a shared file by hash
func (s *LocalStorage) GetSharedFile(hash string) (*SharedFile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	file, exists := s.sharedFiles[hash]
	return file, exists
}

// GetAllSharedHashes returns all shared file hashes
func (s *LocalStorage) GetAllSharedHashes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hashes := make([]string, 0, len(s.sharedFiles))
	for hash := range s.sharedFiles {
		hashes = append(hashes, hash)
	}
	return hashes
}

// StartDownload initializes a new download
func (s *LocalStorage) StartDownload(metadata *protocol.FileMetadata) *DownloadState {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := &DownloadState{
		Metadata:       metadata,
		ChunksReceived: make([]bool, len(metadata.Chunks)),
		TempDir:        filepath.Join(s.baseDir, "temp", metadata.Hash),
		OutputPath:     filepath.Join(s.baseDir, "downloads", metadata.Name),
	}

	os.MkdirAll(state.TempDir, 0755)
	s.downloads[metadata.Hash] = state
	return state
}

// GetDownload retrieves download state by file hash
func (s *LocalStorage) GetDownload(hash string) (*DownloadState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, exists := s.downloads[hash]
	return state, exists
}

// MarkChunkReceived marks a chunk as received
func (s *LocalStorage) MarkChunkReceived(fileHash string, chunkIndex int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if state, exists := s.downloads[fileHash]; exists {
		if chunkIndex < len(state.ChunksReceived) {
			state.ChunksReceived[chunkIndex] = true
		}
	}
}

// IsDownloadComplete checks if all chunks have been received
func (s *LocalStorage) IsDownloadComplete(fileHash string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, exists := s.downloads[fileHash]
	if !exists {
		return false
	}

	for _, received := range state.ChunksReceived {
		if !received {
			return false
		}
	}
	return true
}

// SaveState persists the storage state to disk
func (s *LocalStorage) SaveState() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data := map[string]interface{}{
		"shared_files": s.sharedFiles,
		"downloads":    s.downloads,
	}

	file, err := os.Create(filepath.Join(s.baseDir, "state.json"))
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(data)
}

// GetMissingChunks returns indices of chunks not yet received
func (s *LocalStorage) GetMissingChunks(fileHash string) []int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, exists := s.downloads[fileHash]
	if !exists {
		return nil
	}

	var missing []int
	for i, received := range state.ChunksReceived {
		if !received {
			missing = append(missing, i)
		}
	}
	return missing
}
