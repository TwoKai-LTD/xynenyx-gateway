package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/edwardsims/xynenyx-gateway/config"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status string `json:"status"`
}

// ReadyResponse represents the readiness check response
type ReadyResponse struct {
	Status  string            `json:"status"`
	Services map[string]string `json:"services,omitempty"`
}

// HealthHandler handles health check requests
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status: "healthy",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ReadyHandler handles readiness check requests
func ReadyHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		services := make(map[string]string)
		allHealthy := true

		// Check agent service
		agentHealthy := checkServiceHealth(cfg.AgentServiceURL + "/health")
		if agentHealthy {
			services["agent"] = "healthy"
		} else {
			services["agent"] = "unhealthy"
			allHealthy = false
		}

		// Check RAG service
		ragHealthy := checkServiceHealth(cfg.RAGServiceURL + "/health")
		if ragHealthy {
			services["rag"] = "healthy"
		} else {
			services["rag"] = "unhealthy"
			allHealthy = false
		}

		// Check LLM service
		llmHealthy := checkServiceHealth(cfg.LLMServiceURL + "/health")
		if llmHealthy {
			services["llm"] = "healthy"
		} else {
			services["llm"] = "unhealthy"
			allHealthy = false
		}

		response := ReadyResponse{
			Status:   "ready",
			Services: services,
		}

		statusCode := http.StatusOK
		if !allHealthy {
			response.Status = "not ready"
			statusCode = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)
	}
}

// checkServiceHealth checks if a service is healthy
func checkServiceHealth(url string) bool {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

