package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/p2p-filesharing/distributed-system/pkg/protocol"
)

func TestLocalStorage(t *testing.T) {
	tmpDir := t.TempDir()
	ls, _ := NewLocalStorage(tmpDir)

	// Test AddSharedFile
	t.Run("AddSharedFile", func(t *testing.T) {
		metadata := &protocol.FileMetadata{
			Name: "test.txt",
			Size: 1024,
			Hash: "abc123",
			Chunks: []protocol.ChunkInfo{
				{Index: 0, Hash: "chunk0", Size: 512},
				{Index: 1, Hash: "chunk1", Size: 512},
			},
		}

		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("test content"), 0644)

		ls.AddSharedFile(metadata, testFile)

		// Verify file was added
		sf, exists := ls.GetSharedFile("abc123")
		if !exists {
			t.Fatal("Shared file not found")
		}

		if sf.Metadata.Name != "test.txt" {
			t.Errorf("Expected name test.txt, got %s", sf.Metadata.Name)
		}
	})

	// Test GetAllSharedHashes
	t.Run("GetAllSharedHashes", func(t *testing.T) {
		hashes := ls.GetAllSharedHashes()
		if len(hashes) != 1 {
			t.Errorf("Expected 1 hash, got %d", len(hashes))
		}
	})
}

func TestDownloadState(t *testing.T) {
	tmpDir := t.TempDir()
	ls, _ := NewLocalStorage(tmpDir)

	// Create download state
	t.Run("StartDownload", func(t *testing.T) {
		metadata := &protocol.FileMetadata{
			Name: "download.zip",
			Size: 2048,
			Hash: "def456",
			Chunks: []protocol.ChunkInfo{
				{Index: 0, Hash: "c0", Size: 512},
				{Index: 1, Hash: "c1", Size: 512},
				{Index: 2, Hash: "c2", Size: 512},
				{Index: 3, Hash: "c3", Size: 512},
			},
		}

		state := ls.StartDownload(metadata)

		if state.Metadata.Hash != "def456" {
			t.Errorf("Expected hash def456, got %s", state.Metadata.Hash)
		}

		if len(state.ChunksReceived) != 4 {
			t.Errorf("Expected 4 chunks, got %d", len(state.ChunksReceived))
		}
	})

	// Test MarkChunkReceived
	t.Run("MarkChunkReceived", func(t *testing.T) {
		ls.MarkChunkReceived("def456", 0)
		ls.MarkChunkReceived("def456", 2)

		state, _ := ls.GetDownload("def456")

		if !state.ChunksReceived[0] {
			t.Error("Chunk 0 should be received")
		}

		if state.ChunksReceived[1] {
			t.Error("Chunk 1 should not be received")
		}

		if !state.ChunksReceived[2] {
			t.Error("Chunk 2 should be received")
		}
	})

	// Test GetMissingChunks
	t.Run("GetMissingChunks", func(t *testing.T) {
		missing := ls.GetMissingChunks("def456")

		if len(missing) != 2 {
			t.Errorf("Expected 2 missing chunks, got %d", len(missing))
		}

		// Should be chunks 1 and 3
		if missing[0] != 1 || missing[1] != 3 {
			t.Errorf("Expected [1, 3], got %v", missing)
		}
	})

	// Test IsDownloadComplete
	t.Run("IsDownloadComplete", func(t *testing.T) {
		if ls.IsDownloadComplete("def456") {
			t.Error("Download should not be complete")
		}

		// Complete remaining chunks
		ls.MarkChunkReceived("def456", 1)
		ls.MarkChunkReceived("def456", 3)

		if !ls.IsDownloadComplete("def456") {
			t.Error("Download should be complete")
		}
	})
}
