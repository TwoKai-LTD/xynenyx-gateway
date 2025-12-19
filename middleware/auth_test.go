package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/edwardsims/xynenyx-gateway/config"
)

func TestValidateJWT(t *testing.T) {
	secret := "test-secret-key"
	userID := "user-123"

	// Create a valid token
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Test valid token
	validatedClaims, err := ValidateJWT(tokenString, secret)
	if err != nil {
		t.Fatalf("Expected valid token, got error: %v", err)
	}
	if validatedClaims.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, validatedClaims.UserID)
	}

	// Test invalid secret
	_, err = ValidateJWT(tokenString, "wrong-secret")
	if err == nil {
		t.Error("Expected error for invalid secret")
	}

	// Test expired token
	expiredClaims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredTokenString, _ := expiredToken.SignedString([]byte(secret))

	_, err = ValidateJWT(expiredTokenString, secret)
	if err == nil {
		t.Error("Expected error for expired token")
	}
}

func TestAuthMiddleware(t *testing.T) {
	cfg := &config.Config{
		SupabaseJWTSecret: "test-secret",
	}

	// Create a valid token
	claims := &Claims{
		UserID: "user-123",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(cfg.SupabaseJWTSecret))

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
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "Valid token",
			path:           "/api/agent/chat",
			authHeader:     "Bearer " + tokenString,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing Authorization header",
			path:           "/api/agent/chat",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid token format",
			path:           "/api/agent/chat",
			authHeader:     "Invalid " + tokenString,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Health check bypass",
			path:           "/health",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Ready check bypass",
			path:           "/ready",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()
			middleware := AuthMiddleware(cfg)
			middleware(handler).ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Check X-User-ID header for valid requests
			if tt.expectedStatus == http.StatusOK && tt.path != "/health" && tt.path != "/ready" {
				if req.Header.Get("X-User-ID") != "user-123" {
					t.Errorf("Expected X-User-ID header, got %s", req.Header.Get("X-User-ID"))
				}
			}
		})
	}
}

