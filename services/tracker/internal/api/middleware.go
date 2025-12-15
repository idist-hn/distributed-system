package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// AuthMiddleware validates API key for protected endpoints
type AuthMiddleware struct {
	apiKeys map[string]bool // map of valid API keys
	enabled bool
}

// NewAuthMiddleware creates a new auth middleware
// API keys are loaded from environment variable API_KEYS (comma-separated)
func NewAuthMiddleware() *AuthMiddleware {
	am := &AuthMiddleware{
		apiKeys: make(map[string]bool),
		enabled: false,
	}

	// Load API keys from environment
	keysEnv := os.Getenv("API_KEYS")
	if keysEnv != "" {
		keys := strings.Split(keysEnv, ",")
		for _, key := range keys {
			key = strings.TrimSpace(key)
			if key != "" {
				am.apiKeys[key] = true
				am.enabled = true
			}
		}
		log.Printf("[Auth] Loaded %d API keys", len(am.apiKeys))
	} else {
		log.Println("[Auth] No API keys configured, authentication disabled")
	}

	return am
}

// Middleware returns the HTTP middleware function
func (am *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if not enabled
		if !am.enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth for health check
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Get API key from header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = r.URL.Query().Get("api_key")
		}

		if apiKey == "" || !am.apiKeys[apiKey] {
			http.Error(w, `{"error":"unauthorized","message":"Invalid or missing API key"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*clientLimiter
	rate     int           // requests per window
	window   time.Duration // time window
	enabled  bool
	cleanupT *time.Ticker
}

type clientLimiter struct {
	tokens    int
	lastReset time.Time
}

// NewRateLimiter creates a new rate limiter
// Rate and window are loaded from environment (RATE_LIMIT, RATE_WINDOW_SECONDS)
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*clientLimiter),
		rate:    100,              // default 100 requests
		window:  60 * time.Second, // per minute
		enabled: false,
	}

	// Load from environment
	if rateEnv := os.Getenv("RATE_LIMIT"); rateEnv != "" {
		var rate int
		if _, err := fmt.Sscanf(rateEnv, "%d", &rate); err == nil && rate > 0 {
			rl.rate = rate
			rl.enabled = true
		}
	}

	if windowEnv := os.Getenv("RATE_WINDOW_SECONDS"); windowEnv != "" {
		var window int
		if _, err := fmt.Sscanf(windowEnv, "%d", &window); err == nil && window > 0 {
			rl.window = time.Duration(window) * time.Second
		}
	}

	if rl.enabled {
		log.Printf("[RateLimit] Enabled: %d requests per %v", rl.rate, rl.window)
		// Cleanup old entries every 5 minutes
		rl.cleanupT = time.NewTicker(5 * time.Minute)
		go rl.cleanup()
	} else {
		log.Println("[RateLimit] Disabled")
	}

	return rl
}

func (rl *RateLimiter) cleanup() {
	for range rl.cleanupT.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.window * 2)
		for ip, cl := range rl.clients {
			if cl.lastReset.Before(cutoff) {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// getClientIP extracts client IP from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	// Check X-Real-IP header
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}
	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// Middleware returns the HTTP middleware function
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Skip rate limit for health check
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		clientIP := getClientIP(r)

		rl.mu.Lock()
		cl, exists := rl.clients[clientIP]
		now := time.Now()

		if !exists {
			cl = &clientLimiter{tokens: rl.rate, lastReset: now}
			rl.clients[clientIP] = cl
		}

		// Reset tokens if window has passed
		if now.Sub(cl.lastReset) >= rl.window {
			cl.tokens = rl.rate
			cl.lastReset = now
		}

		if cl.tokens <= 0 {
			rl.mu.Unlock()
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.rate))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(rl.window.Seconds())))
			http.Error(w, `{"error":"rate_limit_exceeded","message":"Too many requests"}`, http.StatusTooManyRequests)
			return
		}

		cl.tokens--
		remaining := cl.tokens
		rl.mu.Unlock()

		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.rate))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		next.ServeHTTP(w, r)
	})
}
