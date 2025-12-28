package handlers

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/edwardsims/xynenyx-gateway/config"
	"github.com/edwardsims/xynenyx-gateway/middleware"
)

// writeErrorWithCORS writes an error response with CORS headers
func writeErrorWithCORS(w http.ResponseWriter, r *http.Request, cfg *config.Config, message string, statusCode int) {
	// Set CORS headers before writing error (matching CORS middleware logic)
	origin := r.Header.Get("Origin")
	if origin != "" && isOriginAllowed(origin, cfg.CORSOrigins) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write([]byte(message))
}

// isOriginAllowed checks if an origin is in the allowed list (duplicated from middleware for use here)
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}
	if len(allowedOrigins) == 0 {
		return true
	}
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	return false
}

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

	// Customize response to strip CORS headers from downstream services
	// and ensure gateway CORS headers are set
	originalModifyResponse := proxy.ModifyResponse
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Strip CORS headers from downstream service response
		resp.Header.Del("Access-Control-Allow-Origin")
		resp.Header.Del("Access-Control-Allow-Credentials")
		resp.Header.Del("Access-Control-Allow-Methods")
		resp.Header.Del("Access-Control-Allow-Headers")
		resp.Header.Del("Access-Control-Expose-Headers")
		resp.Header.Del("Access-Control-Max-Age")

		// Get origin from request (stored in response request)
		origin := resp.Request.Header.Get("Origin")
		if origin != "" && isOriginAllowed(origin, cfg.CORSOrigins) {
			// Set CORS headers on the response
			resp.Header.Set("Access-Control-Allow-Origin", origin)
			resp.Header.Set("Access-Control-Allow-Credentials", "true")
		}

		if originalModifyResponse != nil {
			return originalModifyResponse(resp)
		}
		return nil
	}

	// Customize error handling
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		writeErrorWithCORS(w, r, cfg, "Service unavailable", http.StatusBadGateway)
	}

	// Get circuit breaker for this service
	breaker := circuitBreaker.GetBreaker(serviceName)

	return func(w http.ResponseWriter, r *http.Request) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(r.Context(), cfg.RequestTimeout)
		defer cancel()

		// Create a response writer wrapper to capture status
		statusWriter := &statusResponseWriter{ResponseWriter: w}

		// Execute with circuit breaker protection (Call() handles state checking and transitions)
		err := breaker.Call(func() error {
			// Create a new request with context
			reqWithCtx := r.WithContext(ctx)

			// Log the target URL for debugging
			log.Printf("Proxying request to %s: %s %s", serviceName, targetURL, reqWithCtx.URL.Path)

			// Serve the request
			proxy.ServeHTTP(statusWriter, reqWithCtx)

			// Check for timeout
			if ctx.Err() == context.DeadlineExceeded {
				log.Printf("Request to %s timed out", serviceName)
				return ctx.Err()
			}

			// Only count 5xx errors as failures (not 4xx client errors)
			if statusWriter.statusCode >= 500 {
				log.Printf("Request to %s failed with status %d (target: %s, path: %s)", serviceName, statusWriter.statusCode, targetURL, reqWithCtx.URL.Path)
				return http.ErrAbortHandler
			}
			// 4xx errors are client errors, not service failures - don't count them
			// 2xx and 3xx are successes
			if statusWriter.statusCode < 400 {
				log.Printf("Request to %s succeeded with status %d", serviceName, statusWriter.statusCode)
			}

			return nil
		})

		// Log circuit breaker blocking
		if err != nil && err.Error() == "circuit breaker is open" {
			log.Printf("Circuit breaker blocked request to %s (state: open)", serviceName)
		}

		if err != nil {
			// Check if it's a timeout
			if err == context.DeadlineExceeded {
				if !statusWriter.written {
					writeErrorWithCORS(w, r, cfg, "Request timeout", http.StatusGatewayTimeout)
				}
				return
			}
			// Circuit breaker error
			if err.Error() == "circuit breaker is open" {
				if !statusWriter.written {
					writeErrorWithCORS(w, r, cfg, "Service unavailable", http.StatusServiceUnavailable)
				}
				return
			}
			// Other errors
			if !statusWriter.written {
				writeErrorWithCORS(w, r, cfg, "Service error", http.StatusBadGateway)
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
