package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/edwardsims/xynenyx-gateway/config"
)

type contextKey string

const userIDKey contextKey = "userID"

// AuthMiddleware extracts user ID from X-User-ID header or generates anonymous ID
// This service supports anonymous-only access - no authentication required
func AuthMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip for health checks
			if r.URL.Path == "/health" || r.URL.Path == "/ready" {
				next.ServeHTTP(w, r)
				return
			}

			// Extract user ID from X-User-ID header (anonymous access)
			userID := r.Header.Get("X-User-ID")
			if userID == "" {
				// Generate anonymous user ID from IP address
				userID = "anonymous-" + strings.ReplaceAll(r.RemoteAddr, ":", "-")
			}

			// Set user ID in context and header
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			r.Header.Set("X-User-ID", userID)

			// Continue with request
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts user ID from request context
func GetUserID(r *http.Request) string {
	if userID, ok := r.Context().Value(userIDKey).(string); ok {
		return userID
	}
	return ""
}

