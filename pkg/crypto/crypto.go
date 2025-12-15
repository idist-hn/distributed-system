// Package crypto provides end-to-end encryption for P2P file transfers
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// KeyPair represents an ECDH key pair for key exchange
type KeyPair struct {
	PrivateKey *ecdh.PrivateKey
	PublicKey  *ecdh.PublicKey
}

// GenerateKeyPair generates a new ECDH key pair using P-256 curve
func GenerateKeyPair() (*KeyPair, error) {
	curve := ecdh.P256()
	privateKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  privateKey.PublicKey(),
	}, nil
}

// PublicKeyBytes returns the public key as bytes
func (kp *KeyPair) PublicKeyBytes() []byte {
	return kp.PublicKey.Bytes()
}

// PublicKeyBase64 returns the public key as base64 string
func (kp *KeyPair) PublicKeyBase64() string {
	return base64.StdEncoding.EncodeToString(kp.PublicKeyBytes())
}

// ParsePublicKey parses a public key from bytes
func ParsePublicKey(data []byte) (*ecdh.PublicKey, error) {
	curve := ecdh.P256()
	return curve.NewPublicKey(data)
}

// ParsePublicKeyBase64 parses a public key from base64 string
func ParsePublicKeyBase64(s string) (*ecdh.PublicKey, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return ParsePublicKey(data)
}

// DeriveSharedSecret derives a shared secret using ECDH
func (kp *KeyPair) DeriveSharedSecret(peerPublicKey *ecdh.PublicKey) ([]byte, error) {
	return kp.PrivateKey.ECDH(peerPublicKey)
}

// DeriveSessionKeys derives encryption and MAC keys from shared secret
func DeriveSessionKeys(sharedSecret []byte, info string) (encKey, macKey []byte, err error) {
	// Use HKDF to derive keys
	h := hkdf.New(sha256.New, sharedSecret, nil, []byte(info))

	encKey = make([]byte, 32) // AES-256 key
	macKey = make([]byte, 32) // HMAC key

	if _, err := io.ReadFull(h, encKey); err != nil {
		return nil, nil, err
	}
	if _, err := io.ReadFull(h, macKey); err != nil {
		return nil, nil, err
	}

	return encKey, macKey, nil
}

// Encryptor handles encryption/decryption of data
type Encryptor struct {
	encKey []byte
	macKey []byte
	gcm    cipher.AEAD
}

// NewEncryptor creates a new encryptor with the given key
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &Encryptor{
		encKey: key,
		gcm:    gcm,
	}, nil
}

// NewEncryptorWithKeys creates encryptor with separate enc and mac keys
func NewEncryptorWithKeys(encKey, macKey []byte) (*Encryptor, error) {
	enc, err := NewEncryptor(encKey)
	if err != nil {
		return nil, err
	}
	enc.macKey = macKey
	return enc, nil
}

// Encrypt encrypts data using AES-GCM
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Nonce is prepended to ciphertext
	ciphertext := e.gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using AES-GCM
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := e.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	return e.gcm.Open(nil, nonce, ciphertext, nil)
}

// EncryptChunk encrypts a file chunk with additional metadata
func (e *Encryptor) EncryptChunk(data []byte, chunkIndex int) ([]byte, error) {
	return e.Encrypt(data)
}

// DecryptChunk decrypts a file chunk
func (e *Encryptor) DecryptChunk(data []byte, chunkIndex int) ([]byte, error) {
	return e.Decrypt(data)
}

