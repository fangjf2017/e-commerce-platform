package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ecommerce/feature-management/internal/api"
	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/ecommerce/feature-management/internal/service"
)

// FlagHandler handles CRUD operations for feature flags.
type FlagHandler struct {
	svc *service.FlagService
}

// NewFlagHandler creates a new FlagHandler.
func NewFlagHandler(svc *service.FlagService) *FlagHandler {
	return &FlagHandler{svc: svc}
}

type createFlagRequest struct {
	Key          string          `json:"key"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Type         domain.FlagType `json:"type"`
	DefaultValue json.RawMessage `json:"default_value"`
	Tags         []string        `json:"tags"`
}

type updateFlagRequest struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	DefaultValue json.RawMessage `json:"default_value"`
	Tags         []string        `json:"tags"`
}

// Create handles POST /v1/applications/{appID}/environments/{envID}/flags.
func (h *FlagHandler) Create(w http.ResponseWriter, r *http.Request) {
	appID, err := api.ParseUUID(chi.URLParam(r, "appID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "appID must be a valid UUID")
		return
	}
	envID, err := api.ParseUUID(chi.URLParam(r, "envID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "envID must be a valid UUID")
		return
	}

	var req createFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}

	if req.Key == "" {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "key is required")
		return
	}
	if req.Name == "" {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "name is required")
		return
	}
	if req.Type == "" {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "type is required")
		return
	}
	if len(req.DefaultValue) == 0 {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "default_value is required")
		return
	}

	actor := actorFromRequest(r)

	flag, err := h.svc.Create(r.Context(), service.CreateFlagInput{
		ApplicationID: appID,
		EnvironmentID: envID,
		Key:           req.Key,
		Name:          req.Name,
		Description:   req.Description,
		Type:          req.Type,
		DefaultValue:  []byte(req.DefaultValue),
		Tags:          req.Tags,
		Actor:         actor,
	})
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.JSON(w, http.StatusCreated, api.Response{Data: flag})
}

// List handles GET /v1/applications/{appID}/environments/{envID}/flags.
func (h *FlagHandler) List(w http.ResponseWriter, r *http.Request) {
	appID, err := api.ParseUUID(chi.URLParam(r, "appID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "appID must be a valid UUID")
		return
	}
	envID, err := api.ParseUUID(chi.URLParam(r, "envID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "envID must be a valid UUID")
		return
	}

	limit := api.ParseIntQuery(r, "limit", 20)
	offset := api.ParseIntQuery(r, "offset", 0)

	flags, total, err := h.svc.List(r.Context(), service.ListFlagsInput{
		ApplicationID: appID,
		EnvironmentID: envID,
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.JSON(w, http.StatusOK, api.Response{
		Data: flags,
		Meta: &api.Meta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	})
}

// Get handles GET /v1/applications/{appID}/environments/{envID}/flags/{flagKey}.
func (h *FlagHandler) Get(w http.ResponseWriter, r *http.Request) {
	appID, err := api.ParseUUID(chi.URLParam(r, "appID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "appID must be a valid UUID")
		return
	}
	envID, err := api.ParseUUID(chi.URLParam(r, "envID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "envID must be a valid UUID")
		return
	}
	flagKey := chi.URLParam(r, "flagKey")

	flag, err := h.svc.Get(r.Context(), appID, envID, flagKey)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.JSON(w, http.StatusOK, api.Response{Data: flag})
}

// Update handles PUT /v1/applications/{appID}/environments/{envID}/flags/{flagKey}.
func (h *FlagHandler) Update(w http.ResponseWriter, r *http.Request) {
	appID, err := api.ParseUUID(chi.URLParam(r, "appID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "appID must be a valid UUID")
		return
	}
	envID, err := api.ParseUUID(chi.URLParam(r, "envID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "envID must be a valid UUID")
		return
	}
	flagKey := chi.URLParam(r, "flagKey")

	var req updateFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}

	if req.Name == "" {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "name is required")
		return
	}

	actor := actorFromRequest(r)

	flag, err := h.svc.Update(r.Context(), appID, envID, flagKey, service.UpdateFlagInput{
		Name:         req.Name,
		Description:  req.Description,
		DefaultValue: []byte(req.DefaultValue),
		Tags:         req.Tags,
		Actor:        actor,
	})
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.JSON(w, http.StatusOK, api.Response{Data: flag})
}

// Delete handles DELETE /v1/applications/{appID}/environments/{envID}/flags/{flagKey}.
func (h *FlagHandler) Delete(w http.ResponseWriter, r *http.Request) {
	appID, err := api.ParseUUID(chi.URLParam(r, "appID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "appID must be a valid UUID")
		return
	}
	envID, err := api.ParseUUID(chi.URLParam(r, "envID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "envID must be a valid UUID")
		return
	}
	flagKey := chi.URLParam(r, "flagKey")

	actor := actorFromRequest(r)

	if err := h.svc.Delete(r.Context(), appID, envID, flagKey, actor); err != nil {
		api.HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Enable handles POST /v1/applications/{appID}/environments/{envID}/flags/{flagKey}/enable.
func (h *FlagHandler) Enable(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, domain.FlagStatusActive)
}

// Disable handles POST /v1/applications/{appID}/environments/{envID}/flags/{flagKey}/disable.
func (h *FlagHandler) Disable(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, domain.FlagStatusInactive)
}

// Archive handles POST /v1/applications/{appID}/environments/{envID}/flags/{flagKey}/archive.
func (h *FlagHandler) Archive(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, domain.FlagStatusArchived)
}

func (h *FlagHandler) setStatus(w http.ResponseWriter, r *http.Request, status domain.FlagStatus) {
	appID, err := api.ParseUUID(chi.URLParam(r, "appID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "appID must be a valid UUID")
		return
	}
	envID, err := api.ParseUUID(chi.URLParam(r, "envID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "envID must be a valid UUID")
		return
	}
	flagKey := chi.URLParam(r, "flagKey")
	actor := actorFromRequest(r)

	flag, err := h.svc.SetStatus(r.Context(), appID, envID, flagKey, status, actor)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.JSON(w, http.StatusOK, api.Response{Data: flag})
}
