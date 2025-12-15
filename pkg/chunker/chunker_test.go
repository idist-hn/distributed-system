package chunker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChunkFile(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Write test content (1KB)
	content := make([]byte, 1024)
	for i := range content {
		content[i] = byte(i % 256)
	}
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Chunk with small chunk size for testing
	c := New(256)
	metadata, err := c.ChunkFile(testFile)
	if err != nil {
		t.Fatalf("ChunkFile failed: %v", err)
	}

	if metadata.Name != "test.txt" {
		t.Errorf("Expected name test.txt, got %s", metadata.Name)
	}

	if metadata.Size != 1024 {
		t.Errorf("Expected size 1024, got %d", metadata.Size)
	}

	expectedChunks := 4 // 1024 / 256
	if len(metadata.Chunks) != expectedChunks {
		t.Errorf("Expected %d chunks, got %d", expectedChunks, len(metadata.Chunks))
	}

	// Verify each chunk has a hash
	for i, chunk := range metadata.Chunks {
		if chunk.Hash == "" {
			t.Errorf("Chunk %d has empty hash", i)
		}
		if chunk.Index != i {
			t.Errorf("Chunk %d has wrong index %d", i, chunk.Index)
		}
	}
}

func TestReadWriteChunk(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create file with known content
	content := []byte("Hello World! This is a test file with some content.")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	c := New(16) // Small chunks for testing

	// Read first chunk
	chunk0, err := c.ReadChunk(testFile, 0)
	if err != nil {
		t.Fatalf("ReadChunk failed: %v", err)
	}

	if string(chunk0) != "Hello World! Thi" {
		t.Errorf("Chunk 0 content mismatch: %s", string(chunk0))
	}

	// Test WriteChunk
	outFile := filepath.Join(tmpDir, "output.txt")
	if err := c.WriteChunk(outFile, 0, chunk0); err != nil {
		t.Fatalf("WriteChunk failed: %v", err)
	}

	// Read back and verify
	written, _ := os.ReadFile(outFile)
	if string(written[:len(chunk0)]) != string(chunk0) {
		t.Error("Written chunk doesn't match")
	}
}

func TestGetChunkCount(t *testing.T) {
	c := New(256)

	tests := []struct {
		fileSize int64
		expected int
	}{
		{0, 0},
		{1, 1},
		{256, 1},
		{257, 2},
		{512, 2},
		{1024, 4},
		{1025, 5},
	}

	for _, test := range tests {
		result := c.GetChunkCount(test.fileSize)
		if result != test.expected {
			t.Errorf("GetChunkCount(%d) = %d, expected %d", test.fileSize, result, test.expected)
		}
	}
}
