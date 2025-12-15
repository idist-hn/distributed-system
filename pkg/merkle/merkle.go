// Package merkle provides Merkle tree implementation for data integrity verification
package merkle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

// Node represents a node in the Merkle tree
type Node struct {
	Hash   []byte
	Left   *Node
	Right  *Node
	Parent *Node
	IsLeaf bool
	Data   []byte // Only for leaf nodes
}

// Tree represents a Merkle tree
type Tree struct {
	Root       *Node
	Leaves     []*Node
	MerkleRoot []byte
}

// NewTree creates a Merkle tree from data blocks
func NewTree(dataBlocks [][]byte) (*Tree, error) {
	if len(dataBlocks) == 0 {
		return nil, errors.New("no data blocks provided")
	}

	// Create leaf nodes
	leaves := make([]*Node, len(dataBlocks))
	for i, data := range dataBlocks {
		hash := sha256.Sum256(data)
		leaves[i] = &Node{
			Hash:   hash[:],
			IsLeaf: true,
			Data:   data,
		}
	}

	// Build tree from leaves
	root := buildTree(leaves)

	return &Tree{
		Root:       root,
		Leaves:     leaves,
		MerkleRoot: root.Hash,
	}, nil
}

// NewTreeFromHashes creates a Merkle tree from pre-computed leaf hashes
func NewTreeFromHashes(hashes [][]byte) (*Tree, error) {
	if len(hashes) == 0 {
		return nil, errors.New("no hashes provided")
	}

	leaves := make([]*Node, len(hashes))
	for i, hash := range hashes {
		leaves[i] = &Node{
			Hash:   hash,
			IsLeaf: true,
		}
	}

	root := buildTree(leaves)

	return &Tree{
		Root:       root,
		Leaves:     leaves,
		MerkleRoot: root.Hash,
	}, nil
}

// buildTree builds the Merkle tree recursively
func buildTree(nodes []*Node) *Node {
	if len(nodes) == 1 {
		return nodes[0]
	}

	// Pad with duplicate of last node if odd number
	if len(nodes)%2 != 0 {
		nodes = append(nodes, nodes[len(nodes)-1])
	}

	var parents []*Node
	for i := 0; i < len(nodes); i += 2 {
		left := nodes[i]
		right := nodes[i+1]

		// Concatenate and hash
		combined := append(left.Hash, right.Hash...)
		hash := sha256.Sum256(combined)

		parent := &Node{
			Hash:  hash[:],
			Left:  left,
			Right: right,
		}

		left.Parent = parent
		right.Parent = parent

		parents = append(parents, parent)
	}

	return buildTree(parents)
}

// GetProof returns the Merkle proof for a leaf at given index
func (t *Tree) GetProof(index int) ([]ProofNode, error) {
	if index < 0 || index >= len(t.Leaves) {
		return nil, fmt.Errorf("index out of range: %d", index)
	}

	var proof []ProofNode
	node := t.Leaves[index]

	for node.Parent != nil {
		parent := node.Parent
		var sibling *Node
		var isLeft bool

		if parent.Left == node {
			sibling = parent.Right
			isLeft = false // Sibling is on the right
		} else {
			sibling = parent.Left
			isLeft = true // Sibling is on the left
		}

		proof = append(proof, ProofNode{
			Hash:   sibling.Hash,
			IsLeft: isLeft,
		})

		node = parent
	}

	return proof, nil
}

// ProofNode represents a node in a Merkle proof
type ProofNode struct {
	Hash   []byte `json:"hash"`
	IsLeft bool   `json:"is_left"`
}

// VerifyProof verifies a Merkle proof for given data
func VerifyProof(data []byte, proof []ProofNode, rootHash []byte) bool {
	hash := sha256.Sum256(data)
	currentHash := hash[:]

	for _, p := range proof {
		var combined []byte
		if p.IsLeft {
			combined = append(p.Hash, currentHash...)
		} else {
			combined = append(currentHash, p.Hash...)
		}
		h := sha256.Sum256(combined)
		currentHash = h[:]
	}

	return bytes.Equal(currentHash, rootHash)
}

// VerifyProofWithHash verifies a Merkle proof with pre-computed hash
func VerifyProofWithHash(leafHash []byte, proof []ProofNode, rootHash []byte) bool {
	currentHash := leafHash

	for _, p := range proof {
		var combined []byte
		if p.IsLeft {
			combined = append(p.Hash, currentHash...)
		} else {
			combined = append(currentHash, p.Hash...)
		}
		h := sha256.Sum256(combined)
		currentHash = h[:]
	}

	return bytes.Equal(currentHash, rootHash)
}

// RootHex returns the Merkle root as hex string
func (t *Tree) RootHex() string {
	return hex.EncodeToString(t.MerkleRoot)
}

// GetLeafHash returns the hash of a leaf at given index
func (t *Tree) GetLeafHash(index int) ([]byte, error) {
	if index < 0 || index >= len(t.Leaves) {
		return nil, fmt.Errorf("index out of range: %d", index)
	}
	return t.Leaves[index].Hash, nil
}

// VerifyLeaf verifies a single leaf at given index
func (t *Tree) VerifyLeaf(index int, data []byte) bool {
	if index < 0 || index >= len(t.Leaves) {
		return false
	}

	hash := sha256.Sum256(data)
	return bytes.Equal(hash[:], t.Leaves[index].Hash)
}

// ChunkVerifier provides easy verification for file chunks
type ChunkVerifier struct {
	tree      *Tree
	rootHash  []byte
	proofs    map[int][]ProofNode
	chunkSize int
}

// NewChunkVerifier creates a verifier for file chunks
func NewChunkVerifier(chunkHashes [][]byte) (*ChunkVerifier, error) {
	tree, err := NewTreeFromHashes(chunkHashes)
	if err != nil {
		return nil, err
	}

	// Pre-compute all proofs
	proofs := make(map[int][]ProofNode)
	for i := range chunkHashes {
		proof, _ := tree.GetProof(i)
		proofs[i] = proof
	}

	return &ChunkVerifier{
		tree:     tree,
		rootHash: tree.MerkleRoot,
		proofs:   proofs,
	}, nil
}

// GetRootHash returns the Merkle root hash
func (v *ChunkVerifier) GetRootHash() []byte {
	return v.rootHash
}

// GetRootHex returns the Merkle root as hex string
func (v *ChunkVerifier) GetRootHex() string {
	return hex.EncodeToString(v.rootHash)
}

// VerifyChunk verifies a chunk's integrity
func (v *ChunkVerifier) VerifyChunk(index int, data []byte) bool {
	proof, exists := v.proofs[index]
	if !exists {
		return false
	}
	return VerifyProof(data, proof, v.rootHash)
}

// VerifyChunkHash verifies a chunk using its pre-computed hash
func (v *ChunkVerifier) VerifyChunkHash(index int, chunkHash []byte) bool {
	proof, exists := v.proofs[index]
	if !exists {
		return false
	}
	return VerifyProofWithHash(chunkHash, proof, v.rootHash)
}

// GetProof returns the Merkle proof for a chunk
func (v *ChunkVerifier) GetProof(index int) ([]ProofNode, bool) {
	proof, exists := v.proofs[index]
	return proof, exists
}

// HashData computes SHA256 hash of data
func HashData(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// HashDataHex computes SHA256 hash of data and returns hex string
func HashDataHex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
