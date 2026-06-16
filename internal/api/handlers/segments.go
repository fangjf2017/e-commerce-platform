package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ecommerce/feature-management/internal/api"
	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/ecommerce/feature-management/internal/service"
)

// SegmentHandler handles CRUD operations for audience segments.
type SegmentHandler struct {
	svc *service.SegmentService
}

// NewSegmentHandler creates a new SegmentHandler.
func NewSegmentHandler(svc *service.SegmentService) *SegmentHandler {
	return &SegmentHandler{svc: svc}
}

type segmentRuleReq struct {
	Attribute string                   `json:"attribute"`
	Operator  domain.ConditionOperator `json:"operator"`
	Value     json.RawMessage          `json:"value"`
}

type createSegmentRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Operator    domain.SegmentOperator `json:"operator"`
	Rules       []segmentRuleReq       `json:"rules"`
}

type updateSegmentRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Operator    domain.SegmentOperator `json:"operator"`
	Rules       []segmentRuleReq       `json:"rules"`
}

// Create handles POST /v1/applications/{appID}/segments.
func (h *SegmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	appID, err := api.ParseUUID(chi.URLParam(r, "appID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "appID must be a valid UUID")
		return
	}

	var req createSegmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}

	if req.Name == "" {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "name is required")
		return
	}
	// Default operator to "all" if not provided.
	if req.Operator == "" {
		req.Operator = domain.SegmentOpAll
	}

	actor := actorFromRequest(r)

	segment, err := h.svc.Create(r.Context(), service.CreateSegmentInput{
		ApplicationID: appID,
		Name:          req.Name,
		Description:   req.Description,
		Operator:      req.Operator,
		Rules:         toSegmentRuleInputs(req.Rules),
		Actor:         actor,
	})
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.JSON(w, http.StatusCreated, api.Response{Data: segment})
}

// List handles GET /v1/applications/{appID}/segments.
func (h *SegmentHandler) List(w http.ResponseWriter, r *http.Request) {
	appID, err := api.ParseUUID(chi.URLParam(r, "appID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "appID must be a valid UUID")
		return
	}

	limit := api.ParseIntQuery(r, "limit", 20)
	offset := api.ParseIntQuery(r, "offset", 0)

	segments, total, err := h.svc.List(r.Context(), service.ListSegmentsInput{
		ApplicationID: appID,
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.JSON(w, http.StatusOK, api.Response{
		Data: segments,
		Meta: &api.Meta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	})
}

// Get handles GET /v1/applications/{appID}/segments/{segmentID}.
func (h *SegmentHandler) Get(w http.ResponseWriter, r *http.Request) {
	segmentID, err := api.ParseUUID(chi.URLParam(r, "segmentID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "segmentID must be a valid UUID")
		return
	}

	segment, err := h.svc.Get(r.Context(), segmentID)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.JSON(w, http.StatusOK, api.Response{Data: segment})
}

// Update handles PUT /v1/applications/{appID}/segments/{segmentID}.
func (h *SegmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	segmentID, err := api.ParseUUID(chi.URLParam(r, "segmentID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "segmentID must be a valid UUID")
		return
	}

	var req updateSegmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}

	if req.Name == "" {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "name is required")
		return
	}
	if req.Operator == "" {
		req.Operator = domain.SegmentOpAll
	}

	actor := actorFromRequest(r)

	segment, err := h.svc.Update(r.Context(), segmentID, service.UpdateSegmentInput{
		Name:        req.Name,
		Description: req.Description,
		Operator:    req.Operator,
		Rules:       toSegmentRuleInputs(req.Rules),
		Actor:       actor,
	})
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.JSON(w, http.StatusOK, api.Response{Data: segment})
}

// Delete handles DELETE /v1/applications/{appID}/segments/{segmentID}.
func (h *SegmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	segmentID, err := api.ParseUUID(chi.URLParam(r, "segmentID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "segmentID must be a valid UUID")
		return
	}

	actor := actorFromRequest(r)

	if err := h.svc.Delete(r.Context(), segmentID, actor); err != nil {
		api.HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// toSegmentRuleInputs converts API segment rule requests to service inputs.
func toSegmentRuleInputs(reqs []segmentRuleReq) []service.SegmentRuleInput {
	if len(reqs) == 0 {
		return nil
	}
	out := make([]service.SegmentRuleInput, len(reqs))
	for i, sr := range reqs {
		out[i] = service.SegmentRuleInput{
			Attribute: sr.Attribute,
			Operator:  sr.Operator,
			Value:     []byte(sr.Value),
		}
	}
	return out
}
