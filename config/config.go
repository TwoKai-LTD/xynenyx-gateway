package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the gateway
type Config struct {
	// Supabase
	SupabaseJWTSecret string

	// Service URLs
	AgentServiceURL string
	RAGServiceURL   string
	LLMServiceURL   string

	// Server
	Port string

	// Rate Limiting
	RateLimitRequests int // Requests per minute
	RateLimitBurst     int // Burst size

	// Circuit Breaker
	CircuitBreakerFailures int           // Failures before opening
	CircuitBreakerTimeout  time.Duration // Timeout in seconds

	// Request Timeout
	RequestTimeout time.Duration

	// CORS
	CORSOrigins []string

	// Logging
	LogLevel string
}

// Load loads configuration from environment variables
func Load() *Config {
	cfg := &Config{
		// Supabase (optional - not used for anonymous-only service)
		SupabaseJWTSecret: getEnv("SUPABASE_JWT_SECRET", ""),

		// Service URLs
		AgentServiceURL: getEnv("AGENT_SERVICE_URL", "http://localhost:8001"),
		RAGServiceURL:   getEnv("RAG_SERVICE_URL", "http://localhost:8002"),
		LLMServiceURL:   getEnv("LLM_SERVICE_URL", "http://localhost:8003"),

		// Server
		Port: getEnv("PORT", "8080"),

		// Rate Limiting
		RateLimitRequests: getEnvAsInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitBurst:     getEnvAsInt("RATE_LIMIT_BURST", 10),

		// Circuit Breaker
		CircuitBreakerFailures: getEnvAsInt("CIRCUIT_BREAKER_FAILURES", 5),
		CircuitBreakerTimeout:  time.Duration(getEnvAsInt("CIRCUIT_BREAKER_TIMEOUT", 30)) * time.Second,

		// Request Timeout
		RequestTimeout: time.Duration(getEnvAsInt("REQUEST_TIMEOUT", 30)) * time.Second,

		// CORS
		CORSOrigins: parseCORSOrigins(getEnv("CORS_ORIGINS", "http://localhost:3000,https://xynenyx.com,https://www.xynenyx.com")),

		// Logging
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}

	return cfg
}

// Validate checks that required configuration is present
func (c *Config) Validate() error {
	// SUPABASE_JWT_SECRET is optional for anonymous-only service
	// No validation required
	return nil
}

// getEnv gets an environment variable or returns default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as int or returns default
func getEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

// parseCORSOrigins parses comma-separated CORS origins
func parseCORSOrigins(origins string) []string {
	if origins == "" {
		return []string{}
	}
	parts := strings.Split(origins, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ConfigError represents a configuration error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Field + ": " + e.Message
}

