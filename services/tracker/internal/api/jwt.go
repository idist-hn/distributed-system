package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig holds JWT configuration
type JWTConfig struct {
	SecretKey     string
	Issuer        string
	TokenDuration time.Duration
}

// Claims represents JWT claims
type Claims struct {
	PeerID   string `json:"peer_id"`
	Role     string `json:"role"` // "peer", "admin"
	Hostname string `json:"hostname,omitempty"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT operations
type JWTManager struct {
	config JWTConfig
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey, issuer string, duration time.Duration) *JWTManager {
	return &JWTManager{
		config: JWTConfig{
			SecretKey:     secretKey,
			Issuer:        issuer,
			TokenDuration: duration,
		},
	}
}

// GenerateToken generates a new JWT token
func (m *JWTManager) GenerateToken(peerID, role, hostname string) (string, error) {
	now := time.Now()
	claims := Claims{
		PeerID:   peerID,
		Role:     role,
		Hostname: hostname,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.config.Issuer,
			Subject:   peerID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.config.TokenDuration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.SecretKey))
}

// ValidateToken validates a JWT token and returns claims
func (m *JWTManager) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(m.config.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// ContextKey type for context keys
type ContextKey string

const (
	ClaimsContextKey ContextKey = "claims"
)

// JWTMiddleware creates a middleware that validates JWT tokens
func (m *JWTManager) JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip paths that don't need JWT auth
		skipPaths := []string{"/health", "/dashboard", "/metrics", "/", "/api/auth/login", "/api/auth/register"}
		for _, path := range skipPaths {
			if r.URL.Path == path {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Fall back to API key auth for backward compatibility
			if r.Header.Get("X-API-Key") != "" {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "missing authorization", http.StatusUnauthorized)
			IncrementAuthFailures()
			return
		}

		// Parse Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "invalid authorization format", http.StatusUnauthorized)
			IncrementAuthFailures()
			return
		}

		claims, err := m.ValidateToken(parts[1])
		if err != nil {
			http.Error(w, "invalid token: "+err.Error(), http.StatusUnauthorized)
			IncrementAuthFailures()
			return
		}

		// Add claims to context
		ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetClaimsFromContext retrieves claims from context
func GetClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ClaimsContextKey).(*Claims)
	return claims, ok
}

// RequireRole middleware requires a specific role
func RequireRole(role string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetClaimsFromContext(r.Context())
		if !ok {
			http.Error(w, "no claims in context", http.StatusUnauthorized)
			return
		}
		if claims.Role != role && claims.Role != "admin" {
			http.Error(w, "insufficient permissions", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// AuthResponse represents authentication response
type AuthResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	PeerID    string    `json:"peer_id"`
	Role      string    `json:"role"`
}

// LoginRequest represents login request
type LoginRequest struct {
	PeerID   string `json:"peer_id"`
	Hostname string `json:"hostname"`
	APIKey   string `json:"api_key"` // For initial auth
}

// HandleLogin handles peer login and returns JWT token
func (m *JWTManager) HandleLogin(apiKeys []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Validate API key
		valid := false
		for _, key := range apiKeys {
			if req.APIKey == key {
				valid = true
				break
			}
		}
		if !valid {
			http.Error(w, "invalid API key", http.StatusUnauthorized)
			IncrementAuthFailures()
			return
		}

		// Generate token
		token, err := m.GenerateToken(req.PeerID, "peer", req.Hostname)
		if err != nil {
			http.Error(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		resp := AuthResponse{
			Token:     token,
			ExpiresAt: time.Now().Add(m.config.TokenDuration),
			PeerID:    req.PeerID,
			Role:      "peer",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

