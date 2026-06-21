package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ecommerce/feature-management/internal/api/apiutil"
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
// Supported query parameters: resource_type, action, actor_id,
// after (RFC3339), before (RFC3339), limit, offset.
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	appID, err := apiutil.ParseUUID(chi.URLParam(r, "appID"))
	if err != nil {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_id", "appID must be a valid UUID")
		return
	}

	q := r.URL.Query()

	inp := service.ListAuditInput{
		ApplicationID: appID,
		ResourceType:  q.Get("resource_type"),
		Action:        q.Get("action"),
		ActorID:       q.Get("actor_id"),
	}

	if raw := q.Get("after"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			apiutil.JSONError(w, http.StatusBadRequest, "invalid_input", "after must be an RFC3339 timestamp")
			return
		}
		inp.After = &t
	}

	if raw := q.Get("before"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			apiutil.JSONError(w, http.StatusBadRequest, "invalid_input", "before must be an RFC3339 timestamp")
			return
		}
		inp.Before = &t
	}

	inp.Limit = apiutil.ParseIntQuery(r, "limit", 20)
	inp.Offset = apiutil.ParseIntQuery(r, "offset", 0)

	events, total, err := h.svc.List(r.Context(), inp)
	if err != nil {
		apiutil.HandleError(w, err)
		return
	}

	apiutil.JSON(w, http.StatusOK, apiutil.Response{
		Data: events,
		Meta: &apiutil.Meta{
			Total:  total,
			Limit:  inp.Limit,
			Offset: inp.Offset,
		},
	})
}

// Get handles GET /v1/applications/{appID}/audit/{eventID}.
func (h *AuditHandler) Get(w http.ResponseWriter, r *http.Request) {
	eventID, err := apiutil.ParseUUID(chi.URLParam(r, "eventID"))
	if err != nil {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_id", "eventID must be a valid UUID")
		return
	}

	event, err := h.svc.GetEvent(r.Context(), eventID)
	if err != nil {
		apiutil.HandleError(w, err)
		return
	}

	apiutil.JSON(w, http.StatusOK, apiutil.Response{Data: event})
}
