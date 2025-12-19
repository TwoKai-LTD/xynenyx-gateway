package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggingMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			t.Error("Expected X-Request-ID header")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	middleware := LoggingMiddleware(handler)
	middleware.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Check that X-Request-ID was set
	if req.Header.Get("X-Request-ID") == "" {
		t.Error("Expected X-Request-ID header to be set")
	}
}

func TestLoggingMiddlewareWithUserID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(context.WithValue(req.Context(), userIDKey, "user-123"))
	rr := httptest.NewRecorder()

	middleware := LoggingMiddleware(handler)
	middleware.ServeHTTP(rr, req)

	// The log entry should include user ID
	// We can't easily test the JSON output, but we can verify the middleware runs
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	id2 := generateRequestID()

	if id1 == id2 {
		t.Error("Expected different request IDs")
	}

	if len(id1) == 0 {
		t.Error("Expected non-empty request ID")
	}

	// UUIDs should be 36 characters (with hyphens)
	if len(id1) != 36 {
		t.Errorf("Expected UUID format (36 chars), got %d", len(id1))
	}
}

func TestResponseWriter(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, statusCode: 0}

	// Write should set status to 200 if not set
	rw.Write([]byte("test"))
	if rw.statusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rw.statusCode)
	}

	// WriteHeader should set status
	rw.WriteHeader(http.StatusNotFound)
	if rw.statusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rw.statusCode)
	}

	// Check response body
	if !strings.Contains(rr.Body.String(), "test") {
		t.Error("Expected response body to contain 'test'")
	}
}

