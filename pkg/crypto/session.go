package crypto

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// SessionState represents the state of a secure session
type SessionState int

const (
	SessionStateNew SessionState = iota
	SessionStateHandshaking
	SessionStateEstablished
	SessionStateClosed
)

// SecureSession represents an encrypted P2P session
type SecureSession struct {
	ID           string
	LocalKeyPair *KeyPair
	RemotePublic []byte
	Encryptor    *Encryptor
	State        SessionState
	CreatedAt    time.Time
	LastActivity time.Time
	mu           sync.RWMutex
}

// HandshakeMessage is exchanged during session establishment
type HandshakeMessage struct {
	Type      string `json:"type"` // "hello", "hello_ack", "ready"
	PeerID    string `json:"peer_id"`
	PublicKey string `json:"public_key"` // Base64 encoded
	Nonce     string `json:"nonce,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// NewSecureSession creates a new secure session
func NewSecureSession(id string) (*SecureSession, error) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	return &SecureSession{
		ID:           id,
		LocalKeyPair: keyPair,
		State:        SessionStateNew,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}, nil
}

// CreateHello creates a hello message for handshake
func (s *SecureSession) CreateHello(peerID string) *HandshakeMessage {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.State = SessionStateHandshaking
	s.LastActivity = time.Now()

	return &HandshakeMessage{
		Type:      "hello",
		PeerID:    peerID,
		PublicKey: s.LocalKeyPair.PublicKeyBase64(),
		Timestamp: time.Now().UnixMilli(),
	}
}

// ProcessHello processes a hello message and creates hello_ack
func (s *SecureSession) ProcessHello(msg *HandshakeMessage) (*HandshakeMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Parse peer's public key
	peerPubKey, err := ParsePublicKeyBase64(msg.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}
	s.RemotePublic = peerPubKey.Bytes()

	// Derive shared secret
	sharedSecret, err := s.LocalKeyPair.DeriveSharedSecret(peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive shared secret: %w", err)
	}

	// Derive session keys
	encKey, macKey, err := DeriveSessionKeys(sharedSecret, "p2p-session-v1")
	if err != nil {
		return nil, fmt.Errorf("failed to derive keys: %w", err)
	}

	// Create encryptor
	s.Encryptor, err = NewEncryptorWithKeys(encKey, macKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	s.State = SessionStateHandshaking
	s.LastActivity = time.Now()

	return &HandshakeMessage{
		Type:      "hello_ack",
		PeerID:    msg.PeerID,
		PublicKey: s.LocalKeyPair.PublicKeyBase64(),
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

// CompleteHandshake completes the handshake with hello_ack
func (s *SecureSession) CompleteHandshake(msg *HandshakeMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if msg.Type != "hello_ack" {
		return fmt.Errorf("expected hello_ack, got %s", msg.Type)
	}

	// Parse peer's public key
	peerPubKey, err := ParsePublicKeyBase64(msg.PublicKey)
	if err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}
	s.RemotePublic = peerPubKey.Bytes()

	// Derive shared secret
	sharedSecret, err := s.LocalKeyPair.DeriveSharedSecret(peerPubKey)
	if err != nil {
		return fmt.Errorf("failed to derive shared secret: %w", err)
	}

	// Derive session keys
	encKey, macKey, err := DeriveSessionKeys(sharedSecret, "p2p-session-v1")
	if err != nil {
		return fmt.Errorf("failed to derive keys: %w", err)
	}

	// Create encryptor
	s.Encryptor, err = NewEncryptorWithKeys(encKey, macKey)
	if err != nil {
		return fmt.Errorf("failed to create encryptor: %w", err)
	}

	s.State = SessionStateEstablished
	s.LastActivity = time.Now()

	return nil
}

// IsEstablished returns true if session is ready for encrypted communication
func (s *SecureSession) IsEstablished() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State == SessionStateEstablished
}

// Encrypt encrypts data for transmission
func (s *SecureSession) Encrypt(data []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.State != SessionStateEstablished {
		return nil, fmt.Errorf("session not established")
	}

	s.LastActivity = time.Now()
	return s.Encryptor.Encrypt(data)
}

// Decrypt decrypts received data
func (s *SecureSession) Decrypt(data []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.State != SessionStateEstablished {
		return nil, fmt.Errorf("session not established")
	}

	s.LastActivity = time.Now()
	return s.Encryptor.Decrypt(data)
}

// EncryptJSON encrypts a JSON message
func (s *SecureSession) EncryptJSON(v any) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return s.Encrypt(data)
}

// DecryptJSON decrypts and unmarshals JSON
func (s *SecureSession) DecryptJSON(data []byte, v any) error {
	plaintext, err := s.Decrypt(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(plaintext, v)
}

// Close closes the session
func (s *SecureSession) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = SessionStateClosed
}

// SessionManager manages multiple secure sessions
type SessionManager struct {
	sessions map[string]*SecureSession
	mu       sync.RWMutex
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*SecureSession),
	}
}

// CreateSession creates a new session for a peer
func (m *SessionManager) CreateSession(peerID string) (*SecureSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, err := NewSecureSession(peerID)
	if err != nil {
		return nil, err
	}

	m.sessions[peerID] = session
	return session, nil
}

// GetSession returns a session for a peer
func (m *SessionManager) GetSession(peerID string) (*SecureSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[peerID]
	return session, exists
}

// RemoveSession removes a session
func (m *SessionManager) RemoveSession(peerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, exists := m.sessions[peerID]; exists {
		session.Close()
		delete(m.sessions, peerID)
	}
}

// CleanupStaleSessions removes sessions inactive for given duration
func (m *SessionManager) CleanupStaleSessions(maxAge time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	cutoff := time.Now().Add(-maxAge)

	for peerID, session := range m.sessions {
		session.mu.RLock()
		lastActivity := session.LastActivity
		session.mu.RUnlock()

		if lastActivity.Before(cutoff) {
			session.Close()
			delete(m.sessions, peerID)
			count++
		}
	}

	return count
}
