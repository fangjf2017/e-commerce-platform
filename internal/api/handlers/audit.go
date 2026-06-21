package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ecommerce/feature-management/internal/api"
	"github.com/ecommerce/feature-management/internal/service"
)

// AuditHandler handles audit event endpoints.
type AuditHandler struct {
	svc *service.AuditService
}

// NewAuditHandler creates a new AuditHandler.
func NewAuditHandler(svc *service.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

// List handles GET /v1/applications/{appID}/audit.
// Supported query parameters: resource_type, action, actor_id, after (RFC3339), before (RFC3339), limit, offset.
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	appID, err := api.ParseUUID(chi.URLParam(r, "appID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "appID must be a valid UUID")
		return
	}

	q := r.URL.Query()

	var after *time.Time
	if raw := q.Get("after"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			api.JSONError(w, http.StatusBadRequest, "invalid_input", "after must be an RFC3339 timestamp")
			return
		}
		after = &t
	}

	var before *time.Time
	if raw := q.Get("before"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			api.JSONError(w, http.StatusBadRequest, "invalid_input", "before must be an RFC3339 timestamp")
			return
		}
		before = &t
	}

	limit := api.ParseIntQuery(r, "limit", 20)
	offset := api.ParseIntQuery(r, "offset", 0)

	events, total, err := h.svc.List(r.Context(), service.ListAuditInput{
		ApplicationID: appID,
		ResourceType:  q.Get("resource_type"),
		Action:        q.Get("action"),
		ActorID:       q.Get("actor_id"),
		After:         after,
		Before:        before,
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.JSON(w, http.StatusOK, api.Response{
		Data: events,
		Meta: &api.Meta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	})
}

// Get handles GET /v1/applications/{appID}/audit/{eventID}.
func (h *AuditHandler) Get(w http.ResponseWriter, r *http.Request) {
	eventID, err := api.ParseUUID(chi.URLParam(r, "eventID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "eventID must be a valid UUID")
		return
	}

	event, err := h.svc.GetEvent(r.Context(), eventID)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.JSON(w, http.StatusOK, api.Response{Data: event})
}
