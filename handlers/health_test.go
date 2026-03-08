package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/edwardsims/xynenyx-gateway/config"
	"github.com/edwardsims/xynenyx-gateway/middleware"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	HealthHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	if rr.Header().Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type application/json")
	}
}

func TestReadyHandler(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		AgentServiceURL: server.URL,
		RAGServiceURL:   server.URL,
		LLMServiceURL:   server.URL,
	}

	circuitBreaker := middleware.NewCircuitBreakerManager(5, 30*time.Second)

	req := httptest.NewRequest("GET", "/ready", nil)
	rr := httptest.NewRecorder()

	ReadyHandler(cfg, circuitBreaker)(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	if rr.Header().Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type application/json")
	}
}

func TestCheckServiceHealth(t *testing.T) {
	// Create a healthy server
	healthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthyServer.Close()

	// Test healthy service
	if !checkServiceHealth(healthyServer.URL) {
		t.Error("Expected service to be healthy")
	}

	// Test non-existent service
	if checkServiceHealth("http://localhost:99999/health") {
		t.Error("Expected service to be unhealthy")
	}
}
