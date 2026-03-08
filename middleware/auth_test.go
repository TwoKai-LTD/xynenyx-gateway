package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/edwardsims/xynenyx-gateway/config"
)

func TestAuthMiddleware(t *testing.T) {
	cfg := &config.Config{
		SupabaseJWTSecret: "test-secret",
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r)
		if userID == "" {
			http.Error(w, "No user ID", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	tests := []struct {
		name           string
		path           string
		userIDHeader   string
		expectedStatus int
		expectedUserID string
	}{
		{
			name:           "Valid X-User-ID header",
			path:           "/api/agent/chat",
			userIDHeader:   "user-123",
			expectedStatus: http.StatusOK,
			expectedUserID: "user-123",
		},
		{
			name:           "Missing X-User-ID header (generates anonymous)",
			path:           "/api/agent/chat",
			userIDHeader:   "",
			expectedStatus: http.StatusOK,
			expectedUserID: "", // Will be generated, check it starts with "anonymous-"
		},
		{
			name:           "Health check bypass",
			path:           "/health",
			userIDHeader:   "",
			expectedStatus: http.StatusOK,
			expectedUserID: "",
		},
		{
			name:           "Ready check bypass",
			path:           "/ready",
			userIDHeader:   "",
			expectedStatus: http.StatusOK,
			expectedUserID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.userIDHeader != "" {
				req.Header.Set("X-User-ID", tt.userIDHeader)
			}
			// Set RemoteAddr for anonymous ID generation
			req.RemoteAddr = "127.0.0.1:12345"

			rr := httptest.NewRecorder()
			middleware := AuthMiddleware(cfg)
			middleware(handler).ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Check X-User-ID header for valid requests
			if tt.expectedStatus == http.StatusOK && tt.path != "/health" && tt.path != "/ready" {
				actualUserID := req.Header.Get("X-User-ID")
				if tt.expectedUserID != "" {
					if actualUserID != tt.expectedUserID {
						t.Errorf("Expected X-User-ID header %s, got %s", tt.expectedUserID, actualUserID)
					}
				} else {
					// Should generate anonymous ID
					if !strings.HasPrefix(actualUserID, "anonymous-") {
						t.Errorf("Expected anonymous user ID, got %s", actualUserID)
					}
				}
			}
		})
	}
}
