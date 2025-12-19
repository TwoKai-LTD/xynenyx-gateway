package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/edwardsims/xynenyx-gateway/config"
)

type contextKey string

const userIDKey contextKey = "userID"

// Claims represents JWT claims
type Claims struct {
	UserID string `json:"sub"`
	jwt.RegisteredClaims
}

// AuthMiddleware validates JWT tokens and extracts user ID
func AuthMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for health checks
			if r.URL.Path == "/health" || r.URL.Path == "/ready" {
				next.ServeHTTP(w, r)
				return
			}

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
				return
			}

			// Check Bearer prefix
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Validate token
			claims, err := ValidateJWT(tokenString, cfg.SupabaseJWTSecret)
			if err != nil {
				http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// Set user ID in context and header
			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			r.Header.Set("X-User-ID", claims.UserID)

			// Continue with authenticated request
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ValidateJWT validates a JWT token and returns claims
func ValidateJWT(tokenString, secret string) (*Claims, error) {
	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	// Extract claims
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	// Extract user ID from sub claim
	if claims.UserID == "" {
		// Fallback: try to get from sub in RegisteredClaims
		if claims.RegisteredClaims.Subject != "" {
			claims.UserID = claims.RegisteredClaims.Subject
		} else {
			return nil, jwt.ErrInvalidKey
		}
	}

	return claims, nil
}

// GetUserID extracts user ID from request context
func GetUserID(r *http.Request) string {
	if userID, ok := r.Context().Value(userIDKey).(string); ok {
		return userID
	}
	return ""
}

