package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ecommerce/feature-management/internal/api"
	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/ecommerce/feature-management/internal/service"
)

// RuleHandler handles CRUD operations for rules within a feature flag.
type RuleHandler struct {
	svc *service.FlagService
}

// NewRuleHandler creates a new RuleHandler.
func NewRuleHandler(svc *service.FlagService) *RuleHandler {
	return &RuleHandler{svc: svc}
}

type conditionReq struct {
	Attribute string                   `json:"attribute"`
	Operator  domain.ConditionOperator `json:"operator"`
	Value     json.RawMessage          `json:"value"`
	Negate    bool                     `json:"negate"`
}

type createRuleRequest struct {
	Type          domain.RuleType `json:"type"`
	Priority      int             `json:"priority"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Conditions    []conditionReq  `json:"conditions"`
	SegmentID     *uuid.UUID      `json:"segment_id"`
	RolloutPct    *int            `json:"rollout_pct"`
	ScheduleStart *time.Time      `json:"schedule_start"`
	ScheduleEnd   *time.Time      `json:"schedule_end"`
}

type updateRuleRequest struct {
	Type          domain.RuleType `json:"type"`
	Priority      int             `json:"priority"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Conditions    []conditionReq  `json:"conditions"`
	SegmentID     *uuid.UUID      `json:"segment_id"`
	RolloutPct    *int            `json:"rollout_pct"`
	ScheduleStart *time.Time      `json:"schedule_start"`
	ScheduleEnd   *time.Time      `json:"schedule_end"`
}

type reorderRulesRequest struct {
	RuleIDs []uuid.UUID `json:"rule_ids"`
}

// Create handles POST /v1/applications/{appID}/environments/{envID}/flags/{flagKey}/rules.
func (h *RuleHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	// Fetch the flag to get its ID.
	flag, err := h.svc.Get(r.Context(), appID, envID, flagKey)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	var req createRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}

	if req.Type == "" {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "type is required")
		return
	}

	actor := actorFromRequest(r)

	rule, err := h.svc.CreateRule(r.Context(), service.CreateRuleInput{
		FlagID:        flag.ID,
		Type:          req.Type,
		Priority:      req.Priority,
		Name:          req.Name,
		Description:   req.Description,
		Conditions:    toConditionInputs(req.Conditions),
		SegmentID:     req.SegmentID,
		RolloutPct:    req.RolloutPct,
		ScheduleStart: req.ScheduleStart,
		ScheduleEnd:   req.ScheduleEnd,
		Actor:         actor,
	})
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.JSON(w, http.StatusCreated, api.Response{Data: rule})
}

// List handles GET /v1/applications/{appID}/environments/{envID}/flags/{flagKey}/rules.
func (h *RuleHandler) List(w http.ResponseWriter, r *http.Request) {
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

	api.JSON(w, http.StatusOK, api.Response{
		Data: flag.Rules,
		Meta: &api.Meta{
			Total: len(flag.Rules),
		},
	})
}

// Update handles PUT /v1/applications/{appID}/environments/{envID}/flags/{flagKey}/rules/{ruleID}.
func (h *RuleHandler) Update(w http.ResponseWriter, r *http.Request) {
	ruleID, err := api.ParseUUID(chi.URLParam(r, "ruleID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "ruleID must be a valid UUID")
		return
	}

	var req updateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}

	if req.Type == "" {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "type is required")
		return
	}

	actor := actorFromRequest(r)

	rule, err := h.svc.UpdateRule(r.Context(), ruleID, service.UpdateRuleInput{
		Type:          req.Type,
		Priority:      req.Priority,
		Name:          req.Name,
		Description:   req.Description,
		Conditions:    toConditionInputs(req.Conditions),
		SegmentID:     req.SegmentID,
		RolloutPct:    req.RolloutPct,
		ScheduleStart: req.ScheduleStart,
		ScheduleEnd:   req.ScheduleEnd,
		Actor:         actor,
	})
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.JSON(w, http.StatusOK, api.Response{Data: rule})
}

// Delete handles DELETE /v1/applications/{appID}/environments/{envID}/flags/{flagKey}/rules/{ruleID}.
func (h *RuleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ruleID, err := api.ParseUUID(chi.URLParam(r, "ruleID"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_id", "ruleID must be a valid UUID")
		return
	}

	actor := actorFromRequest(r)

	if err := h.svc.DeleteRule(r.Context(), ruleID, actor); err != nil {
		api.HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Reorder handles PUT /v1/applications/{appID}/environments/{envID}/flags/{flagKey}/rules/reorder.
func (h *RuleHandler) Reorder(w http.ResponseWriter, r *http.Request) {
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

	var req reorderRulesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}

	if len(req.RuleIDs) == 0 {
		api.JSONError(w, http.StatusBadRequest, "invalid_input", "rule_ids is required")
		return
	}

	actor := actorFromRequest(r)

	if err := h.svc.ReorderRules(r.Context(), flag.ID, req.RuleIDs, actor); err != nil {
		api.HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// toConditionInputs converts API condition requests to service inputs.
func toConditionInputs(reqs []conditionReq) []service.ConditionInput {
	if len(reqs) == 0 {
		return nil
	}
	out := make([]service.ConditionInput, len(reqs))
	for i, c := range reqs {
		out[i] = service.ConditionInput{
			Attribute: c.Attribute,
			Operator:  c.Operator,
			Value:     []byte(c.Value),
			Negate:    c.Negate,
		}
	}
	return out
}
