package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ecommerce/feature-management/internal/api/apiutil"
	"github.com/ecommerce/feature-management/internal/api/middleware"
	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/ecommerce/feature-management/internal/service"
)

// ApplicationHandler handles CRUD operations for applications.
type ApplicationHandler struct {
	svc *service.ApplicationService
}

// NewApplicationHandler creates a new ApplicationHandler.
func NewApplicationHandler(svc *service.ApplicationService) *ApplicationHandler {
	return &ApplicationHandler{svc: svc}
}

type createApplicationRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type updateApplicationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Create handles POST /v1/applications.
func (h *ApplicationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createApplicationRequest
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

	app, err := h.svc.Create(r.Context(), service.CreateApplicationInput{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Actor:       actor,
	})
	if err != nil {
		apiutil.HandleError(w, err)
		return
	}

	apiutil.JSON(w, http.StatusCreated, apiutil.Response{Data: app})
}

// List handles GET /v1/applications.
func (h *ApplicationHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := apiutil.ParseIntQuery(r, "limit", 20)
	offset := apiutil.ParseIntQuery(r, "offset", 0)

	apps, total, err := h.svc.List(r.Context(), service.ListApplicationsInput{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		apiutil.HandleError(w, err)
		return
	}

	apiutil.JSON(w, http.StatusOK, apiutil.Response{
		Data: apps,
		Meta: &apiutil.Meta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	})
}

// Get handles GET /v1/applications/{appID}.
func (h *ApplicationHandler) Get(w http.ResponseWriter, r *http.Request) {
	appID, err := apiutil.ParseUUID(chi.URLParam(r, "appID"))
	if err != nil {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_id", "appID must be a valid UUID")
		return
	}

	app, err := h.svc.Get(r.Context(), appID)
	if err != nil {
		apiutil.HandleError(w, err)
		return
	}

	apiutil.JSON(w, http.StatusOK, apiutil.Response{Data: app})
}

// Update handles PATCH /v1/applications/{appID}.
func (h *ApplicationHandler) Update(w http.ResponseWriter, r *http.Request) {
	appID, err := apiutil.ParseUUID(chi.URLParam(r, "appID"))
	if err != nil {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_id", "appID must be a valid UUID")
		return
	}

	var req updateApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}

	actor := actorFromRequest(r)

	app, err := h.svc.Update(r.Context(), appID, service.UpdateApplicationInput{
		Name:        req.Name,
		Description: req.Description,
		Actor:       actor,
	})
	if err != nil {
		apiutil.HandleError(w, err)
		return
	}

	apiutil.JSON(w, http.StatusOK, apiutil.Response{Data: app})
}

// Delete handles DELETE /v1/applications/{appID}.
func (h *ApplicationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	appID, err := apiutil.ParseUUID(chi.URLParam(r, "appID"))
	if err != nil {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_id", "appID must be a valid UUID")
		return
	}

	actor := actorFromRequest(r)

	if err := h.svc.Delete(r.Context(), appID, actor); err != nil {
		apiutil.HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// actorFromRequest extracts the AuditActor from the request context.
// Falls back to an anonymous actor if the middleware did not set one.
func actorFromRequest(r *http.Request) domain.AuditActor {
	v := r.Context().Value(middleware.ActorKey)
	if v == nil {
		return domain.AuditActor{ID: "anonymous", Type: "unknown"}
	}
	actor, ok := v.(domain.AuditActor)
	if !ok {
		return domain.AuditActor{ID: "anonymous", Type: "unknown"}
	}
	return actor
}
