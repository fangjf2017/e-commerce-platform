package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ecommerce/feature-management/internal/api/apiutil"
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

func toSegmentRuleInputs(reqs []segmentRuleReq) []service.SegmentRuleInput {
	out := make([]service.SegmentRuleInput, len(reqs))
	for i, sr := range reqs {
		out[i] = service.SegmentRuleInput{
			Attribute: sr.Attribute,
			Operator:  sr.Operator,
			Value:     sr.Value,
		}
	}
	return out
}

// Create handles POST /v1/applications/{appID}/segments.
func (h *SegmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	appID, err := apiutil.ParseUUID(chi.URLParam(r, "appID"))
	if err != nil {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_id", "appID must be a valid UUID")
		return
	}

	var req createSegmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}
	if req.Name == "" {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_input", "name is required")
		return
	}
	if req.Operator == "" {
		req.Operator = domain.SegmentOpAll
	}

	inp := service.CreateSegmentInput{
		ApplicationID: appID,
		Name:          req.Name,
		Description:   req.Description,
		Operator:      req.Operator,
		Rules:         toSegmentRuleInputs(req.Rules),
		Actor:         actorFromRequest(r),
	}

	result, err := h.svc.Create(r.Context(), inp)
	if err != nil {
		apiutil.HandleError(w, err)
		return
	}

	apiutil.JSON(w, http.StatusCreated, apiutil.Response{Data: result})
}

// List handles GET /v1/applications/{appID}/segments.
func (h *SegmentHandler) List(w http.ResponseWriter, r *http.Request) {
	appID, err := apiutil.ParseUUID(chi.URLParam(r, "appID"))
	if err != nil {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_id", "appID must be a valid UUID")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	segments, total, err := h.svc.List(r.Context(), service.ListSegmentsInput{
		ApplicationID: appID,
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		apiutil.HandleError(w, err)
		return
	}

	apiutil.JSON(w, http.StatusOK, apiutil.Response{
		Data: segments,
		Meta: &apiutil.Meta{Total: total},
	})
}

// Get handles GET /v1/applications/{appID}/segments/{segmentID}.
func (h *SegmentHandler) Get(w http.ResponseWriter, r *http.Request) {
	segmentID, err := apiutil.ParseUUID(chi.URLParam(r, "segmentID"))
	if err != nil {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_id", "segmentID must be a valid UUID")
		return
	}

	segment, err := h.svc.Get(r.Context(), segmentID)
	if err != nil {
		apiutil.HandleError(w, err)
		return
	}

	apiutil.JSON(w, http.StatusOK, apiutil.Response{Data: segment})
}

// Update handles PUT /v1/applications/{appID}/segments/{segmentID}.
func (h *SegmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	segmentID, err := apiutil.ParseUUID(chi.URLParam(r, "segmentID"))
	if err != nil {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_id", "segmentID must be a valid UUID")
		return
	}

	var req updateSegmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}
	if req.Name == "" {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_input", "name is required")
		return
	}
	if req.Operator == "" {
		req.Operator = domain.SegmentOpAll
	}

	inp := service.UpdateSegmentInput{
		Name:        req.Name,
		Description: req.Description,
		Operator:    req.Operator,
		Rules:       toSegmentRuleInputs(req.Rules),
		Actor:       actorFromRequest(r),
	}

	result, err := h.svc.Update(r.Context(), segmentID, inp)
	if err != nil {
		apiutil.HandleError(w, err)
		return
	}

	apiutil.JSON(w, http.StatusOK, apiutil.Response{Data: result})
}

// Delete handles DELETE /v1/applications/{appID}/segments/{segmentID}.
func (h *SegmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	segmentID, err := apiutil.ParseUUID(chi.URLParam(r, "segmentID"))
	if err != nil {
		apiutil.JSONError(w, http.StatusBadRequest, "invalid_id", "segmentID must be a valid UUID")
		return
	}

	actor := actorFromRequest(r)
	if err := h.svc.Delete(r.Context(), segmentID, actor); err != nil {
		apiutil.HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
