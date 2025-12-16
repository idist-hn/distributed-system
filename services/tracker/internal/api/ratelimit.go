package api

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// TokenBucketLimiter manages rate limiting per IP/peer using token bucket
type TokenBucketLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewTokenBucketLimiter creates a new token bucket rate limiter
func NewTokenBucketLimiter(r float64, b int) *TokenBucketLimiter {
	rl := &TokenBucketLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate.Limit(r),
		burst:    b,
	}
	go rl.cleanupLoop()
	return rl
}

// getVisitor returns the rate limiter for an IP
func (rl *TokenBucketLimiter) getVisitor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.rate, rl.burst)
		rl.visitors[ip] = &visitor{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}
	v.lastSeen = time.Now()
	return v.limiter
}

// cleanupLoop removes old entries
func (rl *TokenBucketLimiter) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware creates rate limiting middleware
func (rl *TokenBucketLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get client IP
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.Header.Get("X-Real-IP")
		}
		if ip == "" {
			ip = r.RemoteAddr
		}

		limiter := rl.getVisitor(ip)
		if !limiter.Allow() {
			IncrementRateLimitHits()
			w.Header().Set("Retry-After", "1")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// EndpointRateLimiter provides different limits for different endpoints
type EndpointRateLimiter struct {
	limiters map[string]*TokenBucketLimiter
	default_ *TokenBucketLimiter
}

// NewEndpointRateLimiter creates rate limiter with per-endpoint limits
func NewEndpointRateLimiter() *EndpointRateLimiter {
	return &EndpointRateLimiter{
		limiters: map[string]*TokenBucketLimiter{
			"/api/files/announce":  NewTokenBucketLimiter(10, 20), // 10/s for announces
			"/api/peers/register":  NewTokenBucketLimiter(5, 10),  // 5/s for registrations
			"/api/peers/heartbeat": NewTokenBucketLimiter(1, 5),   // 1/s for heartbeat
		},
		default_: NewTokenBucketLimiter(100, 200), // Default: 100/s
	}
}

// Middleware returns the rate limiting middleware
func (erl *EndpointRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Find specific limiter or use default
		limiter := erl.default_
		for path, l := range erl.limiters {
			if r.URL.Path == path {
				limiter = l
				break
			}
		}

		// Get client IP
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.Header.Get("X-Real-IP")
		}
		if ip == "" {
			ip = r.RemoteAddr
		}

		rl := limiter.getVisitor(ip)
		if !rl.Allow() {
			IncrementRateLimitHits()
			w.Header().Set("Retry-After", "1")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
