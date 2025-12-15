# Merkle Tree Verification

## Tổng quan

**Merkle Tree** cho phép verify integrity của từng chunk mà không cần tải toàn bộ file. Chỉ cần Merkle root hash và proof path.

## Kiến trúc

```
┌─────────────────────────────────────────────────────────────────┐
│                      MERKLE TREE                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│                        ┌───────┐                                │
│                        │ Root  │ ◄─── Merkle Root Hash          │
│                        │ Hash  │                                │
│                        └───┬───┘                                │
│                   ┌────────┴────────┐                           │
│                   │                 │                           │
│               ┌───▼───┐         ┌───▼───┐                       │
│               │ H(01) │         │ H(23) │                       │
│               └───┬───┘         └───┬───┘                       │
│              ┌────┴────┐       ┌────┴────┐                      │
│              │         │       │         │                      │
│          ┌───▼───┐ ┌───▼───┐ ┌───▼───┐ ┌───▼───┐                │
│          │ H(0)  │ │ H(1)  │ │ H(2)  │ │ H(3)  │                │
│          └───┬───┘ └───┬───┘ └───┬───┘ └───┬───┘                │
│              │         │         │         │                    │
│          ┌───▼───┐ ┌───▼───┐ ┌───▼───┐ ┌───▼───┐                │
│          │Chunk 0│ │Chunk 1│ │Chunk 2│ │Chunk 3│ ◄── Data       │
│          └───────┘ └───────┘ └───────┘ └───────┘                │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Merkle Proof

Để verify Chunk 1, chỉ cần:
- Hash của Chunk 1
- H(0) (sibling)
- H(23) (uncle)
- Root Hash

```
┌─────────────────────────────────────────┐
│          MERKLE PROOF for Chunk 1       │
├─────────────────────────────────────────┤
│                                          │
│  Proof = [H(0), H(23)]                  │
│                                          │
│  Verification:                           │
│  1. H1 = SHA256(Chunk 1)                │
│  2. H01 = SHA256(H(0) + H1)   ◄─ H(0)   │
│  3. Root = SHA256(H01 + H(23)) ◄─ H(23) │
│  4. Compare with known Root Hash        │
│                                          │
└─────────────────────────────────────────┘
```

## API

### Tạo Merkle Tree

```go
// Từ raw data blocks
chunks := [][]byte{chunk0, chunk1, chunk2, chunk3}
tree, err := merkle.NewTree(chunks)

// Từ pre-computed hashes
hashes := [][]byte{hash0, hash1, hash2, hash3}
tree, err := merkle.NewTreeFromHashes(hashes)

// Get root hash
rootHash := tree.MerkleRoot
rootHex := tree.RootHex() // hex string
```

### Tạo và Verify Proof

```go
// Get proof for chunk at index
proof, err := tree.GetProof(1) // Proof for chunk 1

// Verify with raw data
isValid := merkle.VerifyProof(chunkData, proof, rootHash)

// Verify with pre-computed hash
isValid := merkle.VerifyProofWithHash(chunkHash, proof, rootHash)
```

### ChunkVerifier (High-level API)

```go
// Create verifier from chunk hashes
chunkHashes := [][]byte{hash0, hash1, hash2, hash3}
verifier, err := merkle.NewChunkVerifier(chunkHashes)

// Get Merkle root
rootHash := verifier.GetRootHash()
rootHex := verifier.GetRootHex()

// Verify chunk data
isValid := verifier.VerifyChunk(1, chunkData)

// Verify chunk hash
isValid := verifier.VerifyChunkHash(1, chunkHash)

// Get proof for sharing
proof, exists := verifier.GetProof(1)
```

## Ví dụ: P2P File Verification

```go
// Sender: Tạo Merkle tree khi share file
chunks := splitFileIntoChunks(file)
tree, _ := merkle.NewTree(chunks)

// Share với peers:
// - Merkle root (trong file metadata)
// - Các chunk data

fileMetadata := FileMetadata{
    Hash:       fileHash,
    MerkleRoot: tree.RootHex(),
    Chunks:     getChunkHashes(chunks),
}

// Receiver: Verify từng chunk khi download
verifier, _ := merkle.NewChunkVerifier(metadata.ChunkHashes)

// Verify mỗi chunk nhận được
for i, chunk := range receivedChunks {
    if !verifier.VerifyChunk(i, chunk) {
        log.Printf("Chunk %d is corrupted!", i)
        // Request chunk again from different peer
    }
}
```

## Utility Functions

```go
// Hash data
hash := merkle.HashData(data)        // []byte
hashHex := merkle.HashDataHex(data)  // string

// Verify single leaf
isValid := tree.VerifyLeaf(index, data)

// Get leaf hash
hash, err := tree.GetLeafHash(index)
```

## Lợi ích

| Feature | Benefit |
|---------|---------|
| **Partial Verification** | Verify 1 chunk mà không cần toàn bộ file |
| **Efficient Proof** | Proof size = O(log n) |
| **Tamper Detection** | Phát hiện dữ liệu bị sửa đổi |
| **Parallel Download** | Verify chunks độc lập |

## Proof Size

| Chunks | Proof Size (nodes) |
|--------|-------------------|
| 4 | 2 |
| 16 | 4 |
| 64 | 6 |
| 256 | 8 |
| 1024 | 10 |

## Lưu ý

1. **Hash algorithm**: Sử dụng SHA-256
2. **Odd chunks**: Tự động duplicate node cuối nếu số lẻ
3. **Pre-computed proofs**: ChunkVerifier pre-compute all proofs
4. **Binary tree**: Mỗi non-leaf node có đúng 2 children

