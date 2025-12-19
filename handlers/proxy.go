package handlers

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/edwardsims/xynenyx-gateway/config"
	"github.com/edwardsims/xynenyx-gateway/middleware"
)

// ProxyHandler creates a reverse proxy handler for a service
func ProxyHandler(cfg *config.Config, serviceName string, circuitBreaker *middleware.CircuitBreakerManager) http.HandlerFunc {
	var targetURL string
	switch serviceName {
	case "agent":
		targetURL = cfg.AgentServiceURL
	case "rag":
		targetURL = cfg.RAGServiceURL
	case "llm":
		targetURL = cfg.LLMServiceURL
	default:
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Unknown service", http.StatusBadRequest)
		}
	}

	// Parse target URL
	target, err := url.Parse(targetURL)
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Invalid target URL", http.StatusInternalServerError)
		}
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Customize director to strip /api/{service} prefix and preserve headers
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Strip /api/{service} prefix from path
		path := req.URL.Path
		prefix := "/api/" + serviceName
		if strings.HasPrefix(path, prefix) {
			newPath := strings.TrimPrefix(path, prefix)
			if newPath == "" {
				newPath = "/"
			}
			req.URL.Path = newPath
		}

		// Preserve important headers
		// X-User-ID and X-Request-ID should already be set by middleware
		req.Header.Set("X-Forwarded-For", req.RemoteAddr)
		req.Header.Set("X-Forwarded-Proto", getScheme(req))
	}

	// Customize error handling
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "Service unavailable", http.StatusBadGateway)
	}

	// Get circuit breaker for this service
	breaker := circuitBreaker.GetBreaker(serviceName)

	return func(w http.ResponseWriter, r *http.Request) {
		// Check circuit breaker state first
		state := breaker.GetState()
		if state == middleware.StateOpen {
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
			return
		}

		// Create context with timeout
		ctx, cancel := context.WithTimeout(r.Context(), cfg.RequestTimeout)
		defer cancel()

		// Create a response writer wrapper to capture status
		statusWriter := &statusResponseWriter{ResponseWriter: w}

		// Execute with circuit breaker protection
		err := breaker.Call(func() error {
			// Create a new request with context
			reqWithCtx := r.WithContext(ctx)

			// Serve the request
			proxy.ServeHTTP(statusWriter, reqWithCtx)

			// Check for timeout
			if ctx.Err() == context.DeadlineExceeded {
				return ctx.Err()
			}

			// Check if we got an error status
			if statusWriter.statusCode >= 500 {
				return http.ErrAbortHandler
			}

			return nil
		})

		if err != nil {
			// Check if it's a timeout
			if err == context.DeadlineExceeded {
				if !statusWriter.written {
					http.Error(w, "Request timeout", http.StatusGatewayTimeout)
				}
				return
			}
			// Circuit breaker error
			if err.Error() == "circuit breaker is open" {
				if !statusWriter.written {
					http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
				}
				return
			}
			// Other errors
			if !statusWriter.written {
				http.Error(w, "Service error", http.StatusBadGateway)
			}
		}
	}
}

// getScheme returns the scheme from the request
func getScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}
	return "http"
}

// statusResponseWriter wraps http.ResponseWriter to capture status code
type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (srw *statusResponseWriter) Write(b []byte) (int, error) {
	srw.written = true
	if srw.statusCode == 0 {
		srw.statusCode = http.StatusOK
	}
	return srw.ResponseWriter.Write(b)
}

func (srw *statusResponseWriter) WriteHeader(statusCode int) {
	srw.written = true
	srw.statusCode = statusCode
	srw.ResponseWriter.WriteHeader(statusCode)
}

