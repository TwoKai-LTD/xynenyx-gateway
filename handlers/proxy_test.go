package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edwardsims/xynenyx-gateway/config"
	"github.com/edwardsims/xynenyx-gateway/middleware"
)

func TestProxyHandler(t *testing.T) {
	// Create a mock backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that /api/agent prefix was stripped
		if r.URL.Path != "/health" {
			t.Errorf("Expected path /health, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer backend.Close()

	cfg := &config.Config{
		AgentServiceURL: backend.URL,
		RAGServiceURL:   backend.URL,
		LLMServiceURL:   backend.URL,
		RequestTimeout:  5,
	}

	circuitBreaker := middleware.NewCircuitBreakerManager(5, 30)

	handler := ProxyHandler(cfg, "agent", circuitBreaker)

	req := httptest.NewRequest("GET", "/api/agent/health", nil)
	rr := httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

