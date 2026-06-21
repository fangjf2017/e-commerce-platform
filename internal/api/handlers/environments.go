package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ecommerce/feature-management/internal/api/apiutil"
	"github.com/ecommerce/feature-management/internal/service"
)

// EnvironmentHandler handles CRUD operations for environments within an application.
type EnvironmentHandler struct {
	svc *service.ApplicationService
}

// NewEnvironmentHandler creates a new EnvironmentHandler.
func NewEnvironmentHandler(svc *service.ApplicationService) *EnvironmentHandler {
	return &EnvironmentHandler{svc: svc}
}

type createEnvironmentRequest struct {
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	RequiresApproval bool   `json:"requires_approval"`
}

type updateEnvironmentRequest struct {
	Name             string `json:"name"`
	RequiresApproval bool   `json:"requires_approval"`
}

// Create handles POST /v1/applications/{appID}/environments.
func (h *EnvironmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	appID, err := apiutil.ParseUUID(chi.URLParam(r, "appID"))
	if err != nil {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_id", "appID must be a valid UUID")
		return
	}

	var req createEnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}

	if req.Name == "" {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_input", "name is required")
		return
	}
	if req.Slug == "" {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_input", "slug is required")
		return
	}

	actor := actorFromRequest(r)

	env, err := h.svc.CreateEnvironment(r.Context(), service.CreateEnvironmentInput{
		ApplicationID:    appID,
		Name:             req.Name,
		Slug:             req.Slug,
		RequiresApproval: req.RequiresApproval,
		Actor:            actor,
	})
	if err != nil {
		apiutil.HandleError(w, err)
		return
	}

	apiutil.JSON(w, http.StatusCreated, apiutil.Response{Data: env})
}

// List handles GET /v1/applications/{appID}/environments.
func (h *EnvironmentHandler) List(w http.ResponseWriter, r *http.Request) {
	appID, err := apiutil.ParseUUID(chi.URLParam(r, "appID"))
	if err != nil {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_id", "appID must be a valid UUID")
		return
	}

	limit := apiutil.ParseIntQuery(r, "limit", 20)
	offset := apiutil.ParseIntQuery(r, "offset", 0)

	envs, total, err := h.svc.ListEnvironments(r.Context(), appID, limit, offset)
	if err != nil {
		apiutil.HandleError(w, err)
		return
	}

	apiutil.JSON(w, http.StatusOK, apiutil.Response{
		Data: envs,
		Meta: &apiutil.Meta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	})
}

// Get handles GET /v1/applications/{appID}/environments/{envID}.
func (h *EnvironmentHandler) Get(w http.ResponseWriter, r *http.Request) {
	envID, err := apiutil.ParseUUID(chi.URLParam(r, "envID"))
	if err != nil {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_id", "envID must be a valid UUID")
		return
	}

	env, err := h.svc.GetEnvironment(r.Context(), envID)
	if err != nil {
		apiutil.HandleError(w, err)
		return
	}

	apiutil.JSON(w, http.StatusOK, apiutil.Response{Data: env})
}

// Update handles PATCH /v1/applications/{appID}/environments/{envID}.
func (h *EnvironmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	envID, err := apiutil.ParseUUID(chi.URLParam(r, "envID"))
	if err != nil {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_id", "envID must be a valid UUID")
		return
	}

	var req updateEnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}

	actor := actorFromRequest(r)

	env, err := h.svc.UpdateEnvironment(r.Context(), envID, service.UpdateEnvironmentInput{
		Name:             req.Name,
		RequiresApproval: req.RequiresApproval,
		Actor:            actor,
	})
	if err != nil {
		apiutil.HandleError(w, err)
		return
	}

	apiutil.JSON(w, http.StatusOK, apiutil.Response{Data: env})
}

// Delete handles DELETE /v1/applications/{appID}/environments/{envID}.
func (h *EnvironmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	envID, err := apiutil.ParseUUID(chi.URLParam(r, "envID"))
	if err != nil {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_id", "envID must be a valid UUID")
		return
	}

	actor := actorFromRequest(r)

	if err := h.svc.DeleteEnvironment(r.Context(), envID, actor); err != nil {
		apiutil.HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
