package merkle

import (
	"bytes"
	"crypto/sha256"
	"testing"
)

func TestNewTree(t *testing.T) {
	tests := []struct {
		name    string
		blocks  [][]byte
		wantErr bool
	}{
		{"single block", [][]byte{[]byte("hello")}, false},
		{"two blocks", [][]byte{[]byte("hello"), []byte("world")}, false},
		{"three blocks", [][]byte{[]byte("a"), []byte("b"), []byte("c")}, false},
		{"four blocks", [][]byte{[]byte("1"), []byte("2"), []byte("3"), []byte("4")}, false},
		{"empty", [][]byte{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, err := NewTree(tt.blocks)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTree() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if tree.Root == nil {
					t.Error("NewTree() Root is nil")
				}
				if len(tree.Leaves) != len(tt.blocks) {
					t.Errorf("NewTree() Leaves = %d, want %d", len(tree.Leaves), len(tt.blocks))
				}
				if len(tree.MerkleRoot) != 32 {
					t.Errorf("NewTree() MerkleRoot length = %d, want 32", len(tree.MerkleRoot))
				}
			}
		})
	}
}

func TestNewTreeFromHashes(t *testing.T) {
	hashes := [][]byte{
		HashData([]byte("block1")),
		HashData([]byte("block2")),
		HashData([]byte("block3")),
	}

	tree, err := NewTreeFromHashes(hashes)
	if err != nil {
		t.Fatalf("NewTreeFromHashes() error = %v", err)
	}

	if len(tree.Leaves) != 3 {
		t.Errorf("Leaves = %d, want 3", len(tree.Leaves))
	}
}

func TestTree_GetProof(t *testing.T) {
	blocks := [][]byte{
		[]byte("block0"),
		[]byte("block1"),
		[]byte("block2"),
		[]byte("block3"),
	}

	tree, err := NewTree(blocks)
	if err != nil {
		t.Fatalf("NewTree() error = %v", err)
	}

	for i := range blocks {
		proof, err := tree.GetProof(i)
		if err != nil {
			t.Errorf("GetProof(%d) error = %v", i, err)
			continue
		}
		// For 4 leaves, proof should have 2 nodes (log2(4) = 2)
		if len(proof) != 2 {
			t.Errorf("GetProof(%d) length = %d, want 2", i, len(proof))
		}
	}

	// Test out of range
	_, err = tree.GetProof(-1)
	if err == nil {
		t.Error("GetProof(-1) should return error")
	}
	_, err = tree.GetProof(4)
	if err == nil {
		t.Error("GetProof(4) should return error")
	}
}

func TestVerifyProof(t *testing.T) {
	blocks := [][]byte{
		[]byte("block0"),
		[]byte("block1"),
		[]byte("block2"),
		[]byte("block3"),
	}

	tree, _ := NewTree(blocks)

	for i, block := range blocks {
		proof, _ := tree.GetProof(i)
		if !VerifyProof(block, proof, tree.MerkleRoot) {
			t.Errorf("VerifyProof(%d) = false, want true", i)
		}
	}

	// Test with wrong data
	proof, _ := tree.GetProof(0)
	if VerifyProof([]byte("wrong data"), proof, tree.MerkleRoot) {
		t.Error("VerifyProof with wrong data should return false")
	}
}

func TestVerifyProofWithHash(t *testing.T) {
	blocks := [][]byte{
		[]byte("block0"),
		[]byte("block1"),
	}

	tree, _ := NewTree(blocks)

	for i, block := range blocks {
		proof, _ := tree.GetProof(i)
		hash := sha256.Sum256(block)
		if !VerifyProofWithHash(hash[:], proof, tree.MerkleRoot) {
			t.Errorf("VerifyProofWithHash(%d) = false, want true", i)
		}
	}
}

func TestTree_RootHex(t *testing.T) {
	tree, _ := NewTree([][]byte{[]byte("test")})
	hex := tree.RootHex()
	if len(hex) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("RootHex() length = %d, want 64", len(hex))
	}
}

func TestTree_VerifyLeaf(t *testing.T) {
	blocks := [][]byte{[]byte("block0"), []byte("block1")}
	tree, _ := NewTree(blocks)

	if !tree.VerifyLeaf(0, blocks[0]) {
		t.Error("VerifyLeaf(0) = false, want true")
	}
	if tree.VerifyLeaf(0, []byte("wrong")) {
		t.Error("VerifyLeaf with wrong data should return false")
	}
	if tree.VerifyLeaf(-1, blocks[0]) {
		t.Error("VerifyLeaf(-1) should return false")
	}
}

func TestChunkVerifier(t *testing.T) {
	chunks := [][]byte{
		[]byte("chunk0"),
		[]byte("chunk1"),
		[]byte("chunk2"),
	}

	hashes := make([][]byte, len(chunks))
	for i, chunk := range chunks {
		hashes[i] = HashData(chunk)
	}

	verifier, err := NewChunkVerifier(hashes)
	if err != nil {
		t.Fatalf("NewChunkVerifier() error = %v", err)
	}

	// Test GetRootHash
	if len(verifier.GetRootHash()) != 32 {
		t.Error("GetRootHash() length != 32")
	}

	// Test GetRootHex
	if len(verifier.GetRootHex()) != 64 {
		t.Error("GetRootHex() length != 64")
	}

	// Test VerifyChunk
	for i, chunk := range chunks {
		if !verifier.VerifyChunk(i, chunk) {
			t.Errorf("VerifyChunk(%d) = false, want true", i)
		}
	}

	// Test VerifyChunkHash
	for i, hash := range hashes {
		if !verifier.VerifyChunkHash(i, hash) {
			t.Errorf("VerifyChunkHash(%d) = false, want true", i)
		}
	}

	// Test GetProof
	proof, exists := verifier.GetProof(0)
	if !exists {
		t.Error("GetProof(0) exists = false, want true")
	}
	if len(proof) == 0 {
		t.Error("GetProof(0) returned empty proof")
	}

	_, exists = verifier.GetProof(100)
	if exists {
		t.Error("GetProof(100) exists = true, want false")
	}
}

func TestHashData(t *testing.T) {
	data := []byte("test data")
	hash := HashData(data)

	if len(hash) != 32 {
		t.Errorf("HashData() length = %d, want 32", len(hash))
	}

	expected := sha256.Sum256(data)
	if !bytes.Equal(hash, expected[:]) {
		t.Error("HashData() != sha256.Sum256()")
	}
}

func TestHashDataHex(t *testing.T) {
	data := []byte("test")
	hex := HashDataHex(data)

	if len(hex) != 64 {
		t.Errorf("HashDataHex() length = %d, want 64", len(hex))
	}
}

func TestTreeConsistency(t *testing.T) {
	// Same data should produce same root
	blocks := [][]byte{[]byte("a"), []byte("b"), []byte("c")}

	tree1, _ := NewTree(blocks)
	tree2, _ := NewTree(blocks)

	if !bytes.Equal(tree1.MerkleRoot, tree2.MerkleRoot) {
		t.Error("Same data should produce same Merkle root")
	}

	// Different data should produce different root
	blocks2 := [][]byte{[]byte("x"), []byte("y"), []byte("z")}
	tree3, _ := NewTree(blocks2)

	if bytes.Equal(tree1.MerkleRoot, tree3.MerkleRoot) {
		t.Error("Different data should produce different Merkle root")
	}
}

