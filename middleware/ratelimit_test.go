package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTokenBucket(t *testing.T) {
	// Create bucket with 10 requests per minute (0.167 per second) and burst of 5
	bucket := NewTokenBucket(10, 5)

	// First 5 requests should be allowed immediately (burst)
	for i := 0; i < 5; i++ {
		allowed, _ := bucket.Allow()
		if !allowed {
			t.Errorf("Request %d should be allowed (burst)", i+1)
		}
	}

	// 6th request should be denied
	allowed, waitTime := bucket.Allow()
	if allowed {
		t.Error("6th request should be denied")
	}
	if waitTime <= 0 {
		t.Error("Wait time should be positive")
	}

	// Wait for tokens to refill
	time.Sleep(2 * time.Second)
	allowed, _ = bucket.Allow()
	if !allowed {
		t.Error("Request should be allowed after waiting")
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(10, 5) // 10 per minute, burst 5

	// Test different users
	user1 := "user-1"
	user2 := "user-2"

	// User 1 should get burst
	for i := 0; i < 5; i++ {
		allowed, _ := limiter.Allow(user1)
		if !allowed {
			t.Errorf("User 1 request %d should be allowed", i+1)
		}
	}

	// User 1 should be rate limited
	allowed, _ := limiter.Allow(user1)
	if allowed {
		t.Error("User 1 should be rate limited")
	}

	// User 2 should still have burst available
	for i := 0; i < 5; i++ {
		allowed, _ := limiter.Allow(user2)
		if !allowed {
			t.Errorf("User 2 request %d should be allowed", i+1)
		}
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	limiter := NewRateLimiter(10, 5)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name           string
		path           string
		userID         string
		requests       int
		expectedStatus int
	}{
		{
			name:           "Health check bypass",
			path:           "/health",
			userID:         "",
			requests:       1,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Burst allowed",
			path:           "/api/agent/chat",
			userID:         "user-1",
			requests:       5,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < tt.requests; i++ {
				req := httptest.NewRequest("GET", tt.path, nil)
				if tt.userID != "" {
					req = req.WithContext(context.WithValue(req.Context(), userIDKey, tt.userID))
				}

				rr := httptest.NewRecorder()
				middleware := RateLimitMiddleware(limiter)
				middleware(handler).ServeHTTP(rr, req)

				if rr.Code != tt.expectedStatus {
					t.Errorf("Request %d: Expected status %d, got %d", i+1, tt.expectedStatus, rr.Code)
				}
			}
		})
	}
}

