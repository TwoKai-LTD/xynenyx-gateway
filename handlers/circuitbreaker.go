package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/edwardsims/xynenyx-gateway/middleware"
)

// CircuitBreakerResetHandler manually resets a circuit breaker
func CircuitBreakerResetHandler(circuitBreaker *middleware.CircuitBreakerManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		service := r.URL.Query().Get("service")
		if service == "" {
			http.Error(w, "service parameter required", http.StatusBadRequest)
			return
		}

		circuitBreaker.Reset(service)

		response := map[string]interface{}{
			"message": "Circuit breaker reset for " + service,
			"service": service,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

