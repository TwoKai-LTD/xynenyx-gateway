package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Save original env
	originalSecret := os.Getenv("SUPABASE_JWT_SECRET")
	originalPort := os.Getenv("PORT")

	// Clean up
	defer func() {
		if originalSecret != "" {
			os.Setenv("SUPABASE_JWT_SECRET", originalSecret)
		} else {
			os.Unsetenv("SUPABASE_JWT_SECRET")
		}
		if originalPort != "" {
			os.Setenv("PORT", originalPort)
		} else {
			os.Unsetenv("PORT")
		}
	}()

	// Test defaults
	os.Unsetenv("SUPABASE_JWT_SECRET")
	os.Unsetenv("PORT")
	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("Expected default port 8080, got %s", cfg.Port)
	}
	if cfg.AgentServiceURL != "http://localhost:8001" {
		t.Errorf("Expected default agent URL, got %s", cfg.AgentServiceURL)
	}
	if cfg.RateLimitRequests != 100 {
		t.Errorf("Expected default rate limit 100, got %d", cfg.RateLimitRequests)
	}

	// Test environment variables
	os.Setenv("PORT", "9090")
	os.Setenv("SUPABASE_JWT_SECRET", "test-secret")
	os.Setenv("RATE_LIMIT_REQUESTS", "200")
	cfg = Load()

	if cfg.Port != "9090" {
		t.Errorf("Expected port 9090, got %s", cfg.Port)
	}
	if cfg.SupabaseJWTSecret != "test-secret" {
		t.Errorf("Expected secret test-secret, got %s", cfg.SupabaseJWTSecret)
	}
	if cfg.RateLimitRequests != 200 {
		t.Errorf("Expected rate limit 200, got %d", cfg.RateLimitRequests)
	}
}

func TestParseCORSOrigins(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"http://localhost:3000", []string{"http://localhost:3000"}},
		{"http://localhost:3000,https://xynenyx.com", []string{"http://localhost:3000", "https://xynenyx.com"}},
		{"http://localhost:3000, https://xynenyx.com", []string{"http://localhost:3000", "https://xynenyx.com"}},
		{"", []string{}},
	}

	for _, tt := range tests {
		result := parseCORSOrigins(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("parseCORSOrigins(%q) length = %d, want %d", tt.input, len(result), len(tt.expected))
			continue
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("parseCORSOrigins(%q)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
			}
		}
	}
}

func TestValidate(t *testing.T) {
	// Save original env
	originalSecret := os.Getenv("SUPABASE_JWT_SECRET")
	defer func() {
		if originalSecret != "" {
			os.Setenv("SUPABASE_JWT_SECRET", originalSecret)
		} else {
			os.Unsetenv("SUPABASE_JWT_SECRET")
		}
	}()

	// Test missing secret
	os.Unsetenv("SUPABASE_JWT_SECRET")
	cfg := Load()
	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for missing SUPABASE_JWT_SECRET")
	}

	// Test valid config
	os.Setenv("SUPABASE_JWT_SECRET", "test-secret")
	cfg = Load()
	err = cfg.Validate()
	if err != nil {
		t.Errorf("Expected no validation error, got %v", err)
	}
}

