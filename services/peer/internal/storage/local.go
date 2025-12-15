package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/p2p-filesharing/distributed-system/pkg/protocol"
)

// DownloadStatus represents the status of a download
type DownloadStatus string

const (
	StatusPending   DownloadStatus = "pending"
	StatusActive    DownloadStatus = "active"
	StatusPaused    DownloadStatus = "paused"
	StatusCompleted DownloadStatus = "completed"
	StatusFailed    DownloadStatus = "failed"
	StatusCancelled DownloadStatus = "cancelled"
)

// LocalStorage manages local file storage for a peer
type LocalStorage struct {
	mu          sync.RWMutex
	baseDir     string
	sharedFiles map[string]*SharedFile    // fileHash -> SharedFile
	downloads   map[string]*DownloadState // fileHash -> DownloadState
	stateFile   string
}

// SharedFile represents a file being shared by this peer
type SharedFile struct {
	Metadata *protocol.FileMetadata `json:"metadata"`
	FilePath string                 `json:"file_path"`
}

// DownloadState tracks the progress of a file download
type DownloadState struct {
	Metadata        *protocol.FileMetadata `json:"metadata"`
	ChunksReceived  []bool                 `json:"chunks_received"`
	TempDir         string                 `json:"temp_dir"`
	OutputPath      string                 `json:"output_path"`
	Status          DownloadStatus         `json:"status"`
	StartedAt       time.Time              `json:"started_at"`
	PausedAt        *time.Time             `json:"paused_at,omitempty"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	BytesDownloaded int64                  `json:"bytes_downloaded"`
	TotalBytes      int64                  `json:"total_bytes"`
	LastError       string                 `json:"last_error,omitempty"`
	RetryCount      int                    `json:"retry_count"`
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

	storage := &LocalStorage{
		baseDir:     baseDir,
		sharedFiles: make(map[string]*SharedFile),
		downloads:   make(map[string]*DownloadState),
		stateFile:   filepath.Join(baseDir, "state.json"),
	}

	// Try to load existing state
	if err := storage.LoadState(); err != nil {
		// Ignore error if file doesn't exist
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return storage, nil
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

// IsFileShared checks if a file (by path) is already being shared
func (s *LocalStorage) IsFileShared(filePath string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, shared := range s.sharedFiles {
		if shared.FilePath == filePath {
			return true
		}
	}
	return false
}

// StartDownload initializes a new download or resumes an existing one
func (s *LocalStorage) StartDownload(metadata *protocol.FileMetadata) *DownloadState {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if download already exists (resume scenario)
	if existing, exists := s.downloads[metadata.Hash]; exists {
		if existing.Status == StatusPaused || existing.Status == StatusFailed {
			existing.Status = StatusActive
			existing.PausedAt = nil
			return existing
		}
		if existing.Status == StatusActive {
			return existing
		}
	}

	state := &DownloadState{
		Metadata:       metadata,
		ChunksReceived: make([]bool, len(metadata.Chunks)),
		TempDir:        filepath.Join(s.baseDir, "temp", metadata.Hash),
		OutputPath:     filepath.Join(s.baseDir, "downloads", metadata.Name),
		Status:         StatusActive,
		StartedAt:      time.Now(),
		TotalBytes:     metadata.Size,
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
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.saveStateUnsafe()
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

// PauseDownload pauses an active download
func (s *LocalStorage) PauseDownload(fileHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, exists := s.downloads[fileHash]
	if !exists {
		return ErrDownloadNotFound
	}

	if state.Status != StatusActive {
		return ErrDownloadNotActive
	}

	now := time.Now()
	state.Status = StatusPaused
	state.PausedAt = &now

	// Save state to disk
	return s.saveStateUnsafe()
}

// ResumeDownload resumes a paused download
func (s *LocalStorage) ResumeDownload(fileHash string) (*DownloadState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, exists := s.downloads[fileHash]
	if !exists {
		return nil, ErrDownloadNotFound
	}

	if state.Status != StatusPaused && state.Status != StatusFailed {
		return nil, ErrDownloadNotPaused
	}

	state.Status = StatusActive
	state.PausedAt = nil

	return state, s.saveStateUnsafe()
}

// CancelDownload cancels and removes a download
func (s *LocalStorage) CancelDownload(fileHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, exists := s.downloads[fileHash]
	if !exists {
		return ErrDownloadNotFound
	}

	state.Status = StatusCancelled

	// Clean up temp files
	if state.TempDir != "" {
		os.RemoveAll(state.TempDir)
	}

	delete(s.downloads, fileHash)
	return s.saveStateUnsafe()
}

// CompleteDownload marks a download as completed
func (s *LocalStorage) CompleteDownload(fileHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, exists := s.downloads[fileHash]
	if !exists {
		return ErrDownloadNotFound
	}

	now := time.Now()
	state.Status = StatusCompleted
	state.CompletedAt = &now

	return s.saveStateUnsafe()
}

// SetDownloadError sets an error on a download
func (s *LocalStorage) SetDownloadError(fileHash string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if state, exists := s.downloads[fileHash]; exists {
		state.Status = StatusFailed
		state.LastError = err.Error()
		state.RetryCount++
		s.saveStateUnsafe()
	}
}

// UpdateDownloadProgress updates bytes downloaded
func (s *LocalStorage) UpdateDownloadProgress(fileHash string, bytesDownloaded int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if state, exists := s.downloads[fileHash]; exists {
		state.BytesDownloaded = bytesDownloaded
	}
}

// GetDownloadProgress returns download progress as percentage
func (s *LocalStorage) GetDownloadProgress(fileHash string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, exists := s.downloads[fileHash]
	if !exists {
		return 0, ErrDownloadNotFound
	}

	totalChunks := len(state.ChunksReceived)
	if totalChunks == 0 {
		return 0, nil
	}

	receivedChunks := 0
	for _, received := range state.ChunksReceived {
		if received {
			receivedChunks++
		}
	}

	return float64(receivedChunks) / float64(totalChunks) * 100, nil
}

// ListDownloads returns all downloads with their status
func (s *LocalStorage) ListDownloads() []*DownloadState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	downloads := make([]*DownloadState, 0, len(s.downloads))
	for _, state := range s.downloads {
		downloads = append(downloads, state)
	}
	return downloads
}

// GetPausedDownloads returns all paused downloads
func (s *LocalStorage) GetPausedDownloads() []*DownloadState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var paused []*DownloadState
	for _, state := range s.downloads {
		if state.Status == StatusPaused {
			paused = append(paused, state)
		}
	}
	return paused
}

// LoadState loads the storage state from disk
func (s *LocalStorage) LoadState() error {
	file, err := os.Open(s.stateFile)
	if err != nil {
		return err
	}
	defer file.Close()

	var data struct {
		SharedFiles map[string]*SharedFile    `json:"shared_files"`
		Downloads   map[string]*DownloadState `json:"downloads"`
	}

	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if data.SharedFiles != nil {
		s.sharedFiles = data.SharedFiles
	}
	if data.Downloads != nil {
		s.downloads = data.Downloads
		// Mark active downloads as paused (since we're restarting)
		for _, state := range s.downloads {
			if state.Status == StatusActive {
				now := time.Now()
				state.Status = StatusPaused
				state.PausedAt = &now
			}
		}
	}

	return nil
}

// saveStateUnsafe saves state without acquiring lock (caller must hold lock)
func (s *LocalStorage) saveStateUnsafe() error {
	data := map[string]any{
		"shared_files": s.sharedFiles,
		"downloads":    s.downloads,
	}

	file, err := os.Create(s.stateFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// Errors
var (
	ErrDownloadNotFound  = errDownloadNotFound{}
	ErrDownloadNotActive = errDownloadNotActive{}
	ErrDownloadNotPaused = errDownloadNotPaused{}
)

type errDownloadNotFound struct{}

func (e errDownloadNotFound) Error() string { return "download not found" }

type errDownloadNotActive struct{}

func (e errDownloadNotActive) Error() string { return "download is not active" }

type errDownloadNotPaused struct{}

func (e errDownloadNotPaused) Error() string { return "download is not paused" }
