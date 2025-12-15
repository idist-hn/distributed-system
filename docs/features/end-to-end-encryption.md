# End-to-End Encryption

## Tổng quan

Tính năng **End-to-End Encryption (E2EE)** đảm bảo dữ liệu được mã hóa khi truyền giữa các peers, chỉ có sender và receiver mới có thể giải mã.

## Kiến trúc

```
┌─────────────────────────────────────────────────────────────────┐
│                    END-TO-END ENCRYPTION                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   Peer A                                          Peer B         │
│   ┌─────────────┐                          ┌─────────────┐      │
│   │ Private Key │                          │ Private Key │      │
│   │ Public Key  │                          │ Public Key  │      │
│   └──────┬──────┘                          └──────┬──────┘      │
│          │                                        │              │
│          │   ┌────────────────────────────┐      │              │
│          └──►│      ECDH Key Exchange      │◄────┘              │
│              │    (P-256 / secp256r1)      │                    │
│              └────────────┬───────────────┘                     │
│                           │                                      │
│                           ▼                                      │
│              ┌────────────────────────────┐                     │
│              │      Shared Secret          │                     │
│              │         (ECDH)              │                     │
│              └────────────┬───────────────┘                     │
│                           │                                      │
│                           ▼                                      │
│              ┌────────────────────────────┐                     │
│              │    HKDF Key Derivation      │                     │
│              │  ┌──────────┐ ┌──────────┐ │                     │
│              │  │ Enc Key  │ │ MAC Key  │ │                     │
│              │  │ (256-bit)│ │ (256-bit)│ │                     │
│              │  └──────────┘ └──────────┘ │                     │
│              └────────────┬───────────────┘                     │
│                           │                                      │
│                           ▼                                      │
│              ┌────────────────────────────┐                     │
│              │      AES-256-GCM            │                     │
│              │   (Authenticated Encryption)│                     │
│              └────────────────────────────┘                     │
└─────────────────────────────────────────────────────────────────┘
```

## Handshake Protocol

```
Peer A                                      Peer B
  │                                           │
  │──────── Hello (PubKey_A) ────────────────►│
  │                                           │
  │◄─────── HelloAck (PubKey_B) ──────────────│
  │                                           │
  │   [Both derive shared secret via ECDH]    │
  │   [Both derive session keys via HKDF]     │
  │                                           │
  │◄═══════ Encrypted Data ══════════════════►│
  │                                           │
```

## Cryptographic Primitives

| Component | Algorithm | Key Size |
|-----------|-----------|----------|
| Key Exchange | ECDH P-256 | 256-bit |
| Key Derivation | HKDF-SHA256 | - |
| Encryption | AES-256-GCM | 256-bit |
| Authentication | GCM Tag | 128-bit |

## API

### Key Pair Generation

```go
// Generate new key pair
keyPair, err := crypto.GenerateKeyPair()

// Export public key
pubKeyBytes := keyPair.PublicKeyBytes()
pubKeyBase64 := keyPair.PublicKeyBase64()

// Parse public key
peerPubKey, err := crypto.ParsePublicKeyBase64(base64String)
```

### Secure Session

```go
// Create session
session, err := crypto.NewSecureSession(peerID)

// Initiator side
hello := session.CreateHello(myPeerID)
// Send hello to peer...
// Receive helloAck...
err := session.CompleteHandshake(helloAck)

// Responder side
helloAck, err := session.ProcessHello(hello)
// Send helloAck...

// Check if ready
if session.IsEstablished() {
    // Encrypt data
    encrypted, err := session.Encrypt(plaintext)
    
    // Decrypt data
    decrypted, err := session.Decrypt(ciphertext)
}
```

### Encryptor (Low-level)

```go
// Create encryptor with key
enc, err := crypto.NewEncryptor(key32bytes)

// Encrypt
ciphertext, err := enc.Encrypt(plaintext)

// Decrypt
plaintext, err := enc.Decrypt(ciphertext)

// For chunks
encrypted, err := enc.EncryptChunk(chunkData, chunkIndex)
decrypted, err := enc.DecryptChunk(encryptedChunk, chunkIndex)
```

### Session Manager

```go
// Create manager
mgr := crypto.NewSessionManager()

// Create session for peer
session, err := mgr.CreateSession(peerID)

// Get existing session
session, exists := mgr.GetSession(peerID)

// Remove session
mgr.RemoveSession(peerID)

// Cleanup stale sessions (e.g., inactive for 1 hour)
count := mgr.CleanupStaleSessions(time.Hour)
```

## HandshakeMessage Format

```json
{
  "type": "hello",
  "peer_id": "abc123...",
  "public_key": "BASE64_ENCODED_PUBLIC_KEY",
  "nonce": "optional_nonce",
  "timestamp": 1705312345678
}
```

## Security Properties

1. **Forward Secrecy**: Mỗi session sử dụng ephemeral key pair
2. **Authentication**: GCM tag đảm bảo integrity và authenticity
3. **Confidentiality**: AES-256 encryption
4. **Replay Protection**: Nonce unique cho mỗi message

## Ví dụ Complete Flow

```go
// Peer A (Initiator)
sessionA, _ := crypto.NewSecureSession("peer-a")
hello := sessionA.CreateHello("peer-a")
sendToPeerB(hello)

helloAck := receiveFromPeerB()
sessionA.CompleteHandshake(helloAck)

// Send encrypted chunk
encryptedChunk, _ := sessionA.Encrypt(chunkData)
sendToPeerB(encryptedChunk)

// Peer B (Responder)
sessionB, _ := crypto.NewSecureSession("peer-b")
hello := receiveFromPeerA()
helloAck, _ := sessionB.ProcessHello(hello)
sendToPeerA(helloAck)

// Receive and decrypt chunk
encryptedChunk := receiveFromPeerA()
chunkData, _ := sessionB.Decrypt(encryptedChunk)
```

## Lưu ý

1. **Key rotation**: Sessions nên được recreate định kỳ
2. **Memory protection**: Keys nên được zeroed sau khi sử dụng
3. **Random quality**: Sử dụng `crypto/rand` cho randomness

