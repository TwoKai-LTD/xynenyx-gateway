package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/edwardsims/xynenyx-gateway/config"
	"github.com/edwardsims/xynenyx-gateway/handlers"
	"github.com/edwardsims/xynenyx-gateway/middleware"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Initialize components
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitBurst)
	circuitBreaker := middleware.NewCircuitBreakerManager(
		cfg.CircuitBreakerFailures,
		cfg.CircuitBreakerTimeout,
	)

	// Create router
	router := mux.NewRouter()

	// Apply middleware in order (outermost first)
	router.Use(middleware.RecoveryMiddleware)
	router.Use(middleware.CORSMiddleware(cfg))
	router.Use(middleware.LoggingMiddleware)
	router.Use(middleware.RateLimitMiddleware(rateLimiter))
	router.Use(middleware.AuthMiddleware(cfg))

	// Health check endpoints (no auth required)
	router.HandleFunc("/health", handlers.HealthHandler).Methods("GET")
	router.HandleFunc("/ready", handlers.ReadyHandler(cfg)).Methods("GET")

	// API routes (auth required via middleware)
	router.PathPrefix("/api/agent").Handler(handlers.ProxyHandler(cfg, "agent", circuitBreaker))
	router.PathPrefix("/api/rag").Handler(handlers.ProxyHandler(cfg, "rag", circuitBreaker))
	router.PathPrefix("/api/llm").Handler(handlers.ProxyHandler(cfg, "llm", circuitBreaker))

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Gateway starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
