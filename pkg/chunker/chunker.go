package chunker

import (
	"encoding/hex"
	"io"
	"os"

	"github.com/p2p-filesharing/distributed-system/pkg/hash"
	"github.com/p2p-filesharing/distributed-system/pkg/merkle"
	"github.com/p2p-filesharing/distributed-system/pkg/protocol"
)

const (
	// DefaultChunkSize is 256KB
	DefaultChunkSize = 256 * 1024
	// MaxChunkSize is 1MB
	MaxChunkSize = 1024 * 1024
)

// Chunker handles file splitting and assembly
type Chunker struct {
	ChunkSize int64
}

// New creates a new Chunker with specified chunk size
func New(chunkSize int64) *Chunker {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	if chunkSize > MaxChunkSize {
		chunkSize = MaxChunkSize
	}
	return &Chunker{ChunkSize: chunkSize}
}

// ChunkFile splits a file into chunks and returns metadata
func (c *Chunker) ChunkFile(filepath string) (*protocol.FileMetadata, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Get file info
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	// Calculate file hash
	fileHash, err := hash.CalculateFile(filepath)
	if err != nil {
		return nil, err
	}

	// Read and hash each chunk
	var chunks []protocol.ChunkInfo
	var chunkHashes [][]byte
	buf := make([]byte, c.ChunkSize)
	index := 0

	// Reset file position
	f.Seek(0, 0)

	for {
		n, err := f.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		chunkData := buf[:n]
		chunkHash := hash.Calculate(chunkData)

		// Store hash bytes for merkle tree
		hashBytes, _ := hex.DecodeString(chunkHash)
		chunkHashes = append(chunkHashes, hashBytes)

		chunks = append(chunks, protocol.ChunkInfo{
			Index: index,
			Hash:  chunkHash,
			Size:  int64(n),
		})
		index++
	}

	// Build Merkle tree and get root
	var merkleRoot string
	if len(chunkHashes) > 0 {
		tree, err := merkle.NewTreeFromHashes(chunkHashes)
		if err == nil {
			merkleRoot = tree.RootHex()
		}
	}

	return &protocol.FileMetadata{
		Name:       stat.Name(),
		Size:       stat.Size(),
		Hash:       fileHash,
		ChunkSize:  c.ChunkSize,
		Chunks:     chunks,
		MerkleRoot: merkleRoot,
	}, nil
}

// ReadChunk reads a specific chunk from a file
func (c *Chunker) ReadChunk(filepath string, index int) ([]byte, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	offset := int64(index) * c.ChunkSize
	_, err = f.Seek(offset, 0)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, c.ChunkSize)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return buf[:n], nil
}

// WriteChunk writes a chunk to a file at the correct position
func (c *Chunker) WriteChunk(filepath string, index int, data []byte) error {
	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	offset := int64(index) * c.ChunkSize
	_, err = f.WriteAt(data, offset)
	return err
}

// GetChunkCount calculates how many chunks a file will have
func (c *Chunker) GetChunkCount(fileSize int64) int {
	count := fileSize / c.ChunkSize
	if fileSize%c.ChunkSize != 0 {
		count++
	}
	return int(count)
}
