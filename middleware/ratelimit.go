package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

// TokenBucket implements token bucket rate limiting
type TokenBucket struct {
	rate       float64   // Tokens per second
	capacity   float64   // Maximum tokens
	tokens     float64   // Current tokens
	lastUpdate time.Time // Last update time
	mu         sync.Mutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(rate, burst int) *TokenBucket {
	return &TokenBucket{
		rate:       float64(rate) / 60.0, // Convert per minute to per second
		capacity:   float64(burst),
		tokens:     float64(burst),
		lastUpdate: time.Now(),
	}
}

// Allow checks if a request is allowed and updates the bucket
func (tb *TokenBucket) Allow() (bool, time.Duration) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastUpdate).Seconds()

	// Add tokens based on elapsed time
	tb.tokens = min(tb.capacity, tb.tokens+elapsed*tb.rate)
	tb.lastUpdate = now

	// Check if we have enough tokens
	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return true, 0
	}

	// Calculate time until next token is available
	needed := 1.0 - tb.tokens
	waitTime := time.Duration(needed/tb.rate) * time.Second
	return false, waitTime
}

// RateLimiter manages rate limiting for multiple users
type RateLimiter struct {
	buckets map[string]*TokenBucket
	rate    int
	burst   int
	mu      sync.RWMutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate, burst int) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*TokenBucket),
		rate:    rate,
		burst:   burst,
	}
}

// Allow checks if a request for a user is allowed
func (rl *RateLimiter) Allow(userID string) (bool, time.Duration) {
	// Get or create bucket for user
	rl.mu.RLock()
	bucket, exists := rl.buckets[userID]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		// Double-check after acquiring write lock
		bucket, exists = rl.buckets[userID]
		if !exists {
			bucket = NewTokenBucket(rl.rate, rl.burst)
			rl.buckets[userID] = bucket
		}
		rl.mu.Unlock()
	}

	return bucket.Allow()
}

// RateLimitMiddleware implements rate limiting middleware
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting for health checks
			if r.URL.Path == "/health" || r.URL.Path == "/ready" {
				next.ServeHTTP(w, r)
				return
			}

			// Get user ID from context (set by auth middleware)
			userID := GetUserID(r)
			if userID == "" {
				// If no user ID, use IP address as fallback
				userID = r.RemoteAddr
			}

			// Check rate limit
			allowed, waitTime := limiter.Allow(userID)
			if !allowed {
				w.Header().Set("Retry-After", formatRetryAfter(waitTime))
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// formatRetryAfter formats wait time as seconds for Retry-After header
func formatRetryAfter(d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds < 1 {
		return "1"
	}
	return strconv.Itoa(seconds)
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

