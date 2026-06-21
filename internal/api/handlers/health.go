package handlers

import (
	"encoding/json"
	"net/http"
)

// HealthHandler serves liveness and readiness probes.
type HealthHandler struct{}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Liveness always returns 200 {"status":"ok"}.
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Readiness returns 200 when the service is ready to handle traffic.
// Detailed infra checks (DB, Redis) are handled at the infrastructure level;
// this endpoint simply confirms the process is alive and accepting connections.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}
