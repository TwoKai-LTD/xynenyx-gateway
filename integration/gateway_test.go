package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/edwardsims/xynenyx-gateway/config"
	"github.com/edwardsims/xynenyx-gateway/handlers"
	"github.com/edwardsims/xynenyx-gateway/middleware"
	"github.com/gorilla/mux"
)

// TestGatewayIntegration tests the full gateway flow
func TestGatewayIntegration(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create mock backend service
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("X-User-ID") == "" {
			t.Error("Expected X-User-ID header")
		}
		if r.Header.Get("X-Request-ID") == "" {
			t.Error("Expected X-Request-ID header")
		}

		// Verify path prefix was stripped
		if r.URL.Path == "/api/agent/health" {
			t.Error("Expected /api/agent prefix to be stripped")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer backend.Close()

	cfg := &config.Config{
		SupabaseJWTSecret: "test-secret",
		AgentServiceURL:   backend.URL,
		RAGServiceURL:     backend.URL,
		LLMServiceURL:     backend.URL,
		RequestTimeout:    5 * time.Second,
		RateLimitRequests: 100,
		RateLimitBurst:    10,
		CORSOrigins:       []string{"http://localhost:3000"},
	}

	// Use X-User-ID header for testing (anonymous access)
	testUserID := "test-user-123"

	// Setup router with middleware
	router := mux.NewRouter()
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitBurst)
	circuitBreaker := middleware.NewCircuitBreakerManager(5, 30*time.Second)

	router.Use(middleware.RecoveryMiddleware)
	router.Use(middleware.CORSMiddleware(cfg))
	router.Use(middleware.LoggingMiddleware)
	router.Use(middleware.RateLimitMiddleware(rateLimiter))
	router.Use(middleware.AuthMiddleware(cfg))

	router.HandleFunc("/health", handlers.HealthHandler).Methods("GET")
	router.HandleFunc("/ready", handlers.ReadyHandler(cfg, circuitBreaker)).Methods("GET")
	router.PathPrefix("/api/agent").Handler(handlers.ProxyHandler(cfg, "agent", circuitBreaker))

	// Test health check (no auth)
	t.Run("HealthCheck", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})

	// Test authenticated request (using X-User-ID header)
	t.Run("AuthenticatedRequest", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/agent/health", nil)
		req.Header.Set("X-User-ID", testUserID)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})

	// Test request without X-User-ID (should generate anonymous ID)
	t.Run("AnonymousRequest", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/agent/health", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Should succeed with anonymous user ID generated
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})

	// Test CORS preflight
	t.Run("CORSPreflight", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/agent/health", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "GET")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		if rr.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
			t.Error("Expected CORS header")
		}
	})
}

// TestRateLimitingIntegration tests rate limiting with multiple requests
func TestRateLimitingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg := &config.Config{
		SupabaseJWTSecret: "test-secret",
		RateLimitRequests: 5, // Very low for testing
		RateLimitBurst:    2,
	}

	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitBurst)

	// First 2 requests should be allowed (burst)
	for i := 0; i < 2; i++ {
		allowed, _ := rateLimiter.Allow("test-user")
		if !allowed {
			t.Errorf("Request %d should be allowed (burst)", i+1)
		}
	}

	// Next requests should be rate limited
	allowed, _ := rateLimiter.Allow("test-user")
	if allowed {
		t.Error("Request should be rate limited after burst")
	}
}

// TestCircuitBreakerIntegration tests circuit breaker with service failures
func TestCircuitBreakerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create a failing backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Service error", http.StatusInternalServerError)
	}))
	defer backend.Close()

	circuitBreaker := middleware.NewCircuitBreakerManager(3, 1*time.Second)

	// Fail 3 times to open circuit
	for i := 0; i < 3; i++ {
		breaker := circuitBreaker.GetBreaker("agent")
		breaker.Call(func() error {
			// Simulate failure
			return http.ErrServerClosed
		})
	}

	// Circuit should be open
	if circuitBreaker.GetState("agent") != middleware.StateOpen {
		t.Error("Expected circuit to be open")
	}
}
