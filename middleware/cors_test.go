package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edwardsims/xynenyx-gateway/config"
)

func TestCORSMiddleware(t *testing.T) {
	cfg := &config.Config{
		CORSOrigins: []string{"http://localhost:3000", "https://xynenyx.com"},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name           string
		origin         string
		method         string
		expectedOrigin string
		expectedStatus int
	}{
		{
			name:           "Allowed origin",
			origin:         "http://localhost:3000",
			method:         "GET",
			expectedOrigin: "http://localhost:3000",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Preflight request",
			origin:         "http://localhost:3000",
			method:         "OPTIONS",
			expectedOrigin: "http://localhost:3000",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Disallowed origin",
			origin:         "http://evil.com",
			method:         "GET",
			expectedOrigin: "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "No origin",
			origin:         "",
			method:         "GET",
			expectedOrigin: "",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			rr := httptest.NewRecorder()
			middleware := CORSMiddleware(cfg)
			middleware(handler).ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			actualOrigin := rr.Header().Get("Access-Control-Allow-Origin")
			if actualOrigin != tt.expectedOrigin {
				t.Errorf("Expected origin %s, got %s", tt.expectedOrigin, actualOrigin)
			}

			if tt.method == "OPTIONS" {
				methods := rr.Header().Get("Access-Control-Allow-Methods")
				if methods == "" {
					t.Error("Expected Access-Control-Allow-Methods header")
				}
			}
		})
	}
}

func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		name     string
		origin   string
		allowed  []string
		expected bool
	}{
		{
			name:     "Exact match",
			origin:   "http://localhost:3000",
			allowed:  []string{"http://localhost:3000"},
			expected: true,
		},
		{
			name:     "No match",
			origin:   "http://evil.com",
			allowed:  []string{"http://localhost:3000"},
			expected: false,
		},
		{
			name:     "Empty origin",
			origin:   "",
			allowed:  []string{"http://localhost:3000"},
			expected: false,
		},
		{
			name:     "Empty allowed list",
			origin:   "http://localhost:3000",
			allowed:  []string{},
			expected: true, // Allows all in development
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOriginAllowed(tt.origin, tt.allowed)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

