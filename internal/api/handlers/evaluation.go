package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/ecommerce/feature-management/internal/api"
	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/ecommerce/feature-management/internal/service"
)

// EvaluationHandler handles flag evaluation endpoints.
type EvaluationHandler struct {
	svc *service.EvaluationService
}

// NewEvaluationHandler creates a new EvaluationHandler.
func NewEvaluationHandler(svc *service.EvaluationService) *EvaluationHandler {
	return &EvaluationHandler{svc: svc}
}

type evaluateRequest struct {
	ApplicationID uuid.UUID              `json:"application_id"`
	EnvironmentID uuid.UUID              `json:"environment_id"`
	FlagKey       string                 `json:"flag_key"`
	EntityID      string                 `json:"entity_id"`
	EntityType    string                 `json:"entity_type"`
	Attributes    map[string]interface{} `json:"attributes"`
}

type batchEvaluateRequest struct {
	ApplicationID uuid.UUID              `json:"application_id"`
	EnvironmentID uuid.UUID              `json:"environment_id"`
	EntityID      string                 `json:"entity_id"`
	EntityType    string                 `json:"entity_type"`
	Attributes    map[string]interface{} `json:"attributes"`
	FlagKeys      []string               `json:"flag_keys"`
}

// Evaluate handles POST /v1/evaluate.
func (h *EvaluationHandler) Evaluate(w http.ResponseWriter, r *http.Request) {
	var req evaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}

	if req.ApplicationID == uuid.Nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "application_id is required")
		return
	}
	if req.EnvironmentID == uuid.Nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "environment_id is required")
		return
	}
	if req.FlagKey == "" {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "flag_key is required")
		return
	}
	if req.EntityID == "" {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "entity_id is required")
		return
	}

	requestID := api.RequestIDFromContext(r.Context())

	ec := domain.EvaluationContext{
		FlagKey:       req.FlagKey,
		ApplicationID: req.ApplicationID,
		EnvironmentID: req.EnvironmentID,
		EntityID:      req.EntityID,
		EntityType:    req.EntityType,
		Attributes:    req.Attributes,
		RequestID:     requestID,
		Timestamp:     time.Now().UTC(),
	}

	result, err := h.svc.Evaluate(r.Context(), ec)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.JSON(w, http.StatusOK, api.Response{Data: result})
}

// BatchEvaluate handles POST /v1/evaluate/batch.
func (h *EvaluationHandler) BatchEvaluate(w http.ResponseWriter, r *http.Request) {
	var req batchEvaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}

	if req.ApplicationID == uuid.Nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "application_id is required")
		return
	}
	if req.EnvironmentID == uuid.Nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "environment_id is required")
		return
	}
	if req.EntityID == "" {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "entity_id is required")
		return
	}

	batchReq := domain.BatchEvaluationRequest{
		ApplicationID: req.ApplicationID,
		EnvironmentID: req.EnvironmentID,
		EntityID:      req.EntityID,
		EntityType:    req.EntityType,
		Attributes:    req.Attributes,
		FlagKeys:      req.FlagKeys,
	}

	results, err := h.svc.BatchEvaluate(r.Context(), batchReq)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.JSON(w, http.StatusOK, api.Response{
		Data: results,
		Meta: &api.Meta{
			Total: len(results),
		},
	})
}

// DryRun handles POST /v1/evaluate/dry-run.
// It behaves identically to Evaluate but bypasses any caching side-effects.
func (h *EvaluationHandler) DryRun(w http.ResponseWriter, r *http.Request) {
	var req evaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}

	if req.ApplicationID == uuid.Nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "application_id is required")
		return
	}
	if req.EnvironmentID == uuid.Nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "environment_id is required")
		return
	}
	if req.FlagKey == "" {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "flag_key is required")
		return
	}
	if req.EntityID == "" {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "entity_id is required")
		return
	}

	requestID := api.RequestIDFromContext(r.Context())

	ec := domain.EvaluationContext{
		FlagKey:       req.FlagKey,
		ApplicationID: req.ApplicationID,
		EnvironmentID: req.EnvironmentID,
		EntityID:      req.EntityID,
		EntityType:    req.EntityType,
		Attributes:    req.Attributes,
		RequestID:     requestID,
		Timestamp:     time.Now().UTC(),
	}

	result, err := h.svc.DryRun(r.Context(), ec)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.JSON(w, http.StatusOK, api.Response{Data: result})
}
