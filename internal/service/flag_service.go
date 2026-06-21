package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ecommerce/feature-management/internal/cache"
	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/ecommerce/feature-management/internal/repository"
	"github.com/google/uuid"
)

// --- Input types ---

// CreateFlagInput holds the fields needed to create a feature flag.
type CreateFlagInput struct {
	ApplicationID uuid.UUID
	EnvironmentID uuid.UUID
	Key           string
	Name          string
	Description   string
	Type          domain.FlagType
	DefaultValue  []byte
	Tags          []string
	Actor         domain.AuditActor
}

// ListFlagsInput holds filter and pagination options for listing flags.
type ListFlagsInput struct {
	ApplicationID uuid.UUID
	EnvironmentID uuid.UUID
	Status        *domain.FlagStatus
	Tags          []string
	Limit         int
	Offset        int
}

// UpdateFlagInput holds mutable fields for a flag update.
type UpdateFlagInput struct {
	Name         string
	Description  string
	DefaultValue []byte
	Tags         []string
	Actor        domain.AuditActor
}

// ConditionInput represents a single condition in a rule.
type ConditionInput struct {
	Attribute string
	Operator  domain.ConditionOperator
	Value     []byte
	Negate    bool
}

// CreateRuleInput holds all fields needed to create a rule.
type CreateRuleInput struct {
	FlagID        uuid.UUID
	Type          domain.RuleType
	Priority      int
	Name          string
	Description   string
	Conditions    []ConditionInput
	SegmentID     *uuid.UUID
	RolloutPct    *int
	ScheduleStart *time.Time
	ScheduleEnd   *time.Time
	Actor         domain.AuditActor
}

// UpdateRuleInput holds all fields needed to update a rule.
type UpdateRuleInput struct {
	Type          domain.RuleType
	Priority      int
	Name          string
	Description   string
	Conditions    []ConditionInput
	SegmentID     *uuid.UUID
	RolloutPct    *int
	ScheduleStart *time.Time
	ScheduleEnd   *time.Time
	Actor         domain.AuditActor
}

// --- Service ---

// FlagService manages feature flags and their rules.
type FlagService struct {
	flags       *repository.FlagRepo
	rules       *repository.RuleRepo
	audit       *repository.AuditRepo
	cache       *cache.TieredCache
	invalidator *cache.Invalidator
}

// NewFlagService creates a new FlagService.
func NewFlagService(
	flags *repository.FlagRepo,
	rules *repository.RuleRepo,
	audit *repository.AuditRepo,
	c *cache.TieredCache,
	inv *cache.Invalidator,
) *FlagService {
	return &FlagService{
		flags:       flags,
		rules:       rules,
		audit:       audit,
		cache:       c,
		invalidator: inv,
	}
}

// Create creates a new flag, saves an audit event, and invalidates cache.
func (s *FlagService) Create(ctx context.Context, inp CreateFlagInput) (*domain.FeatureFlag, error) {
	flag := &domain.FeatureFlag{
		ApplicationID: inp.ApplicationID,
		EnvironmentID: inp.EnvironmentID,
		Key:           inp.Key,
		Name:          inp.Name,
		Description:   inp.Description,
		Type:          inp.Type,
		Status:        domain.FlagStatusInactive,
		DefaultValue:  inp.DefaultValue,
		Tags:          inp.Tags,
	}

	if err := s.flags.Create(ctx, flag); err != nil {
		return nil, fmt.Errorf("create flag: %w", err)
	}

	after, _ := json.Marshal(flag)
	auditEvent := &domain.AuditEvent{
		ID:            uuid.New(),
		ApplicationID: flag.ApplicationID,
		EnvironmentID: &flag.EnvironmentID,
		Action:        domain.AuditActionFlagCreated,
		ResourceType:  "feature_flag",
		ResourceID:    flag.ID,
		ResourceKey:   flag.Key,
		Actor:         inp.Actor,
		After:         after,
		OccurredAt:    time.Now().UTC(),
	}
	_ = s.audit.Create(ctx, auditEvent)
	_ = s.invalidator.PublishFlagInvalidation(ctx, flag.ApplicationID, flag.EnvironmentID, flag.Key, string(domain.AuditActionFlagCreated), flag.Version)

	return flag, nil
}

// Get returns a flag by app/env/key (bypasses cache — reads directly from DB).
func (s *FlagService) Get(ctx context.Context, appID, envID uuid.UUID, key string) (*domain.FeatureFlag, error) {
	flag, err := s.flags.GetByKey(ctx, appID, envID, key)
	if err != nil {
		return nil, fmt.Errorf("get flag: %w", err)
	}
	return flag, nil
}

// GetByID returns a flag by ID.
func (s *FlagService) GetByID(ctx context.Context, id uuid.UUID) (*domain.FeatureFlag, error) {
	flag, err := s.flags.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get flag by id: %w", err)
	}
	return flag, nil
}

// List returns paginated flags.
func (s *FlagService) List(ctx context.Context, inp ListFlagsInput) ([]*domain.FeatureFlag, int, error) {
	limit := inp.Limit
	if limit <= 0 {
		limit = 20
	}
	flags, total, err := s.flags.List(ctx, inp.ApplicationID, inp.EnvironmentID, inp.Status, inp.Tags, limit, inp.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list flags: %w", err)
	}
	return flags, total, nil
}

// Update updates a flag, writes an audit event, and invalidates cache.
func (s *FlagService) Update(ctx context.Context, appID, envID uuid.UUID, key string, inp UpdateFlagInput) (*domain.FeatureFlag, error) {
	flag, err := s.flags.GetByKey(ctx, appID, envID, key)
	if err != nil {
		return nil, fmt.Errorf("get flag for update: %w", err)
	}
	before, _ := json.Marshal(flag)

	flag.Name = inp.Name
	flag.Description = inp.Description
	flag.DefaultValue = inp.DefaultValue
	flag.Tags = inp.Tags

	if err := s.flags.Update(ctx, flag); err != nil {
		return nil, fmt.Errorf("update flag: %w", err)
	}

	after, _ := json.Marshal(flag)
	auditEvent := &domain.AuditEvent{
		ID:            uuid.New(),
		ApplicationID: flag.ApplicationID,
		EnvironmentID: &flag.EnvironmentID,
		Action:        domain.AuditActionFlagUpdated,
		ResourceType:  "feature_flag",
		ResourceID:    flag.ID,
		ResourceKey:   flag.Key,
		Actor:         inp.Actor,
		Before:        before,
		After:         after,
		OccurredAt:    time.Now().UTC(),
	}
	_ = s.audit.Create(ctx, auditEvent)
	_ = s.invalidator.PublishFlagInvalidation(ctx, flag.ApplicationID, flag.EnvironmentID, flag.Key, string(domain.AuditActionFlagUpdated), flag.Version)

	return flag, nil
}

// Delete soft-deletes a flag, writes an audit event, and invalidates cache.
func (s *FlagService) Delete(ctx context.Context, appID, envID uuid.UUID, key string, actor domain.AuditActor) error {
	flag, err := s.flags.GetByKey(ctx, appID, envID, key)
	if err != nil {
		return fmt.Errorf("get flag for delete: %w", err)
	}
	before, _ := json.Marshal(flag)

	if err := s.flags.Delete(ctx, flag.ID); err != nil {
		return fmt.Errorf("delete flag: %w", err)
	}

	auditEvent := &domain.AuditEvent{
		ID:            uuid.New(),
		ApplicationID: flag.ApplicationID,
		EnvironmentID: &flag.EnvironmentID,
		Action:        domain.AuditActionFlagDeleted,
		ResourceType:  "feature_flag",
		ResourceID:    flag.ID,
		ResourceKey:   flag.Key,
		Actor:         actor,
		Before:        before,
		OccurredAt:    time.Now().UTC(),
	}
	_ = s.audit.Create(ctx, auditEvent)
	_ = s.invalidator.PublishFlagInvalidation(ctx, flag.ApplicationID, flag.EnvironmentID, flag.Key, string(domain.AuditActionFlagDeleted), flag.Version)

	return nil
}

// SetStatus changes flag status, writes an audit event, and invalidates cache.
func (s *FlagService) SetStatus(ctx context.Context, appID, envID uuid.UUID, key string, status domain.FlagStatus, actor domain.AuditActor) (*domain.FeatureFlag, error) {
	flag, err := s.flags.GetByKey(ctx, appID, envID, key)
	if err != nil {
		return nil, fmt.Errorf("get flag for status change: %w", err)
	}
	before, _ := json.Marshal(flag)

	updated, err := s.flags.UpdateStatus(ctx, flag.ID, status)
	if err != nil {
		return nil, fmt.Errorf("update flag status: %w", err)
	}

	after, _ := json.Marshal(updated)
	auditEvent := &domain.AuditEvent{
		ID:            uuid.New(),
		ApplicationID: updated.ApplicationID,
		EnvironmentID: &updated.EnvironmentID,
		Action:        domain.AuditActionFlagStatusChanged,
		ResourceType:  "feature_flag",
		ResourceID:    updated.ID,
		ResourceKey:   updated.Key,
		Actor:         actor,
		Before:        before,
		After:         after,
		Metadata:      map[string]string{"status": string(status)},
		OccurredAt:    time.Now().UTC(),
	}
	_ = s.audit.Create(ctx, auditEvent)
	_ = s.invalidator.PublishFlagInvalidation(ctx, updated.ApplicationID, updated.EnvironmentID, updated.Key, string(domain.AuditActionFlagStatusChanged), updated.Version)

	return updated, nil
}

// CreateRule adds a rule to a flag.
func (s *FlagService) CreateRule(ctx context.Context, inp CreateRuleInput) (*domain.Rule, error) {
	rule := &domain.Rule{
		FlagID:        inp.FlagID,
		Type:          inp.Type,
		Priority:      inp.Priority,
		Name:          inp.Name,
		Description:   inp.Description,
		SegmentID:     inp.SegmentID,
		RolloutPct:    inp.RolloutPct,
		RolloutSalt:   uuid.New(),
		ScheduleStart: inp.ScheduleStart,
		ScheduleEnd:   inp.ScheduleEnd,
		Enabled:       true,
		Conditions:    toConditions(inp.Conditions),
		Variants:      []domain.VariantAllocation{},
	}

	if err := s.rules.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("create rule: %w", err)
	}

	after, _ := json.Marshal(rule)
	flag, _ := s.flags.GetByID(ctx, inp.FlagID)
	auditEvent := &domain.AuditEvent{
		ID:           uuid.New(),
		Action:       domain.AuditActionRuleCreated,
		ResourceType: "rule",
		ResourceID:   rule.ID,
		ResourceKey:  rule.Name,
		Actor:        inp.Actor,
		After:        after,
		OccurredAt:   time.Now().UTC(),
	}
	if flag != nil {
		auditEvent.ApplicationID = flag.ApplicationID
		auditEvent.EnvironmentID = &flag.EnvironmentID
		_ = s.invalidator.PublishFlagInvalidation(ctx, flag.ApplicationID, flag.EnvironmentID, flag.Key, string(domain.AuditActionRuleCreated), flag.Version)
	}
	_ = s.audit.Create(ctx, auditEvent)

	return rule, nil
}

// UpdateRule updates a rule by ID.
func (s *FlagService) UpdateRule(ctx context.Context, ruleID uuid.UUID, inp UpdateRuleInput) (*domain.Rule, error) {
	existing, err := s.rules.GetByID(ctx, ruleID)
	if err != nil {
		return nil, fmt.Errorf("get rule for update: %w", err)
	}
	before, _ := json.Marshal(existing)

	existing.Type = inp.Type
	existing.Priority = inp.Priority
	existing.Name = inp.Name
	existing.Description = inp.Description
	existing.SegmentID = inp.SegmentID
	existing.RolloutPct = inp.RolloutPct
	existing.ScheduleStart = inp.ScheduleStart
	existing.ScheduleEnd = inp.ScheduleEnd
	existing.Conditions = toConditions(inp.Conditions)

	if err := s.rules.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update rule: %w", err)
	}

	after, _ := json.Marshal(existing)
	flag, _ := s.flags.GetByID(ctx, existing.FlagID)
	auditEvent := &domain.AuditEvent{
		ID:           uuid.New(),
		Action:       domain.AuditActionRuleUpdated,
		ResourceType: "rule",
		ResourceID:   existing.ID,
		ResourceKey:  existing.Name,
		Actor:        inp.Actor,
		Before:       before,
		After:        after,
		OccurredAt:   time.Now().UTC(),
	}
	if flag != nil {
		auditEvent.ApplicationID = flag.ApplicationID
		auditEvent.EnvironmentID = &flag.EnvironmentID
		_ = s.invalidator.PublishFlagInvalidation(ctx, flag.ApplicationID, flag.EnvironmentID, flag.Key, string(domain.AuditActionRuleUpdated), flag.Version)
	}
	_ = s.audit.Create(ctx, auditEvent)

	return existing, nil
}

// DeleteRule deletes a rule by ID.
func (s *FlagService) DeleteRule(ctx context.Context, ruleID uuid.UUID, actor domain.AuditActor) error {
	existing, err := s.rules.GetByID(ctx, ruleID)
	if err != nil {
		return fmt.Errorf("get rule for delete: %w", err)
	}
	before, _ := json.Marshal(existing)

	if err := s.rules.Delete(ctx, ruleID); err != nil {
		return fmt.Errorf("delete rule: %w", err)
	}

	flag, _ := s.flags.GetByID(ctx, existing.FlagID)
	auditEvent := &domain.AuditEvent{
		ID:           uuid.New(),
		Action:       domain.AuditActionRuleDeleted,
		ResourceType: "rule",
		ResourceID:   existing.ID,
		ResourceKey:  existing.Name,
		Actor:        actor,
		Before:       before,
		OccurredAt:   time.Now().UTC(),
	}
	if flag != nil {
		auditEvent.ApplicationID = flag.ApplicationID
		auditEvent.EnvironmentID = &flag.EnvironmentID
		_ = s.invalidator.PublishFlagInvalidation(ctx, flag.ApplicationID, flag.EnvironmentID, flag.Key, string(domain.AuditActionRuleDeleted), flag.Version)
	}
	_ = s.audit.Create(ctx, auditEvent)

	return nil
}

// ReorderRules sets priorities based on order of ruleIDs.
func (s *FlagService) ReorderRules(ctx context.Context, flagID uuid.UUID, ruleIDs []uuid.UUID, actor domain.AuditActor) error {
	if err := s.rules.ReorderByPriority(ctx, flagID, ruleIDs); err != nil {
		return fmt.Errorf("reorder rules: %w", err)
	}

	flag, _ := s.flags.GetByID(ctx, flagID)
	if flag != nil {
		_ = s.invalidator.PublishFlagInvalidation(ctx, flag.ApplicationID, flag.EnvironmentID, flag.Key, string(domain.AuditActionFlagUpdated), flag.Version)
	}
	return nil
}

// toConditions converts ConditionInput slice to domain.Condition slice.
func toConditions(inputs []ConditionInput) []domain.Condition {
	if len(inputs) == 0 {
		return []domain.Condition{}
	}
	out := make([]domain.Condition, len(inputs))
	for i, c := range inputs {
		out[i] = domain.Condition{
			Attribute: c.Attribute,
			Operator:  c.Operator,
			Value:     c.Value,
			Negate:    c.Negate,
		}
	}
	return out
}
