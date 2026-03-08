package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	RequestID  string    `json:"request_id"`
	UserID     string    `json:"user_id,omitempty"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	StatusCode int       `json:"status_code"`
	DurationMS int64     `json:"duration_ms"`
	Error      string    `json:"error,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// LoggingMiddleware implements structured JSON logging
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Generate or get request ID
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		r.Header.Set("X-Request-ID", requestID)

		// Create response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(rw, r)

		// Calculate duration
		duration := time.Since(start)

		// Get user ID from context
		userID := GetUserID(r)

		// Create log entry
		entry := LogEntry{
			RequestID:  requestID,
			UserID:     userID,
			Method:     r.Method,
			Path:       r.URL.Path,
			StatusCode: rw.statusCode,
			DurationMS: duration.Milliseconds(),
			Timestamp:  time.Now(),
		}

		// Add error if status code indicates error
		if rw.statusCode >= 400 {
			entry.Error = http.StatusText(rw.statusCode)
		}

		// Log as JSON
		logJSON(entry)
	})
}

// generateRequestID generates a UUID for request correlation
func generateRequestID() string {
	return uuid.New().String()
}

// logJSON logs a log entry as JSON
func logJSON(entry LogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Failed to marshal log entry: %v", err)
		return
	}
	log.Println(string(data))
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

