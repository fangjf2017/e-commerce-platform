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

// SegmentRuleInput represents a single rule within a segment.
type SegmentRuleInput struct {
	Attribute string
	Operator  domain.ConditionOperator
	Value     []byte
}

// CreateSegmentInput holds all fields needed to create a segment.
type CreateSegmentInput struct {
	ApplicationID uuid.UUID
	Name          string
	Description   string
	Operator      domain.SegmentOperator
	Rules         []SegmentRuleInput
	Actor         domain.AuditActor
}

// ListSegmentsInput holds filter and pagination for listing segments.
type ListSegmentsInput struct {
	ApplicationID uuid.UUID
	Limit         int
	Offset        int
}

// UpdateSegmentInput holds mutable fields for a segment update.
type UpdateSegmentInput struct {
	Name        string
	Description string
	Operator    domain.SegmentOperator
	Rules       []SegmentRuleInput
	Actor       domain.AuditActor
}

// SegmentService manages user segments.
type SegmentService struct {
	segments    *repository.SegmentRepo
	audit       *repository.AuditRepo
	invalidator *cache.Invalidator
}

// NewSegmentService creates a new SegmentService.
func NewSegmentService(segments *repository.SegmentRepo, audit *repository.AuditRepo, inv *cache.Invalidator) *SegmentService {
	return &SegmentService{
		segments:    segments,
		audit:       audit,
		invalidator: inv,
	}
}

// Create creates a new segment.
func (s *SegmentService) Create(ctx context.Context, inp CreateSegmentInput) (*domain.Segment, error) {
	if inp.Operator == "" {
		inp.Operator = domain.SegmentOpAll
	}
	seg := &domain.Segment{
		ApplicationID: inp.ApplicationID,
		Name:          inp.Name,
		Description:   inp.Description,
		Operator:      inp.Operator,
		Rules:         toSegmentRulesDomain(uuid.Nil, inp.Rules),
	}

	if err := s.segments.Create(ctx, seg); err != nil {
		return nil, fmt.Errorf("create segment: %w", err)
	}

	after, _ := json.Marshal(seg)
	auditEvent := &domain.AuditEvent{
		ID:            uuid.New(),
		ApplicationID: seg.ApplicationID,
		Action:        domain.AuditActionSegmentCreated,
		ResourceType:  "segment",
		ResourceID:    seg.ID,
		ResourceKey:   seg.Name,
		Actor:         inp.Actor,
		After:         after,
		OccurredAt:    time.Now().UTC(),
	}
	_ = s.audit.Create(ctx, auditEvent)
	_ = s.invalidator.PublishSegmentInvalidation(ctx, seg.ApplicationID, seg.ID)

	return seg, nil
}

// Get retrieves a segment by ID (single UUID — appID inferred from the segment record).
func (s *SegmentService) Get(ctx context.Context, segID uuid.UUID) (*domain.Segment, error) {
	// Use uuid.Nil as appID since the handler doesn't scope by app in Get.
	seg, err := s.segments.GetByID(ctx, uuid.Nil, segID)
	if err != nil {
		return nil, fmt.Errorf("get segment: %w", err)
	}
	return seg, nil
}

// List returns paginated segments for an application.
func (s *SegmentService) List(ctx context.Context, inp ListSegmentsInput) ([]*domain.Segment, int, error) {
	segs, err := s.segments.List(ctx, inp.ApplicationID)
	if err != nil {
		return nil, 0, fmt.Errorf("list segments: %w", err)
	}
	total := len(segs)

	limit := inp.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := inp.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= total {
		return []*domain.Segment{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return segs[offset:end], total, nil
}

// Update updates a segment's fields and rules by ID.
func (s *SegmentService) Update(ctx context.Context, segID uuid.UUID, inp UpdateSegmentInput) (*domain.Segment, error) {
	// Fetch to get ApplicationID
	existing, err := s.segments.GetByID(ctx, uuid.Nil, segID)
	if err != nil {
		return nil, fmt.Errorf("get segment for update: %w", err)
	}
	before, _ := json.Marshal(existing)

	if inp.Operator == "" {
		inp.Operator = domain.SegmentOpAll
	}
	existing.Name = inp.Name
	existing.Description = inp.Description
	existing.Operator = inp.Operator
	existing.Rules = toSegmentRulesDomain(segID, inp.Rules)

	if err := s.segments.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update segment: %w", err)
	}

	after, _ := json.Marshal(existing)
	auditEvent := &domain.AuditEvent{
		ID:            uuid.New(),
		ApplicationID: existing.ApplicationID,
		Action:        domain.AuditActionSegmentUpdated,
		ResourceType:  "segment",
		ResourceID:    existing.ID,
		ResourceKey:   existing.Name,
		Actor:         inp.Actor,
		Before:        before,
		After:         after,
		OccurredAt:    time.Now().UTC(),
	}
	_ = s.audit.Create(ctx, auditEvent)
	_ = s.invalidator.PublishSegmentInvalidation(ctx, existing.ApplicationID, existing.ID)

	return existing, nil
}

// Delete removes a segment by ID.
func (s *SegmentService) Delete(ctx context.Context, id uuid.UUID, actor domain.AuditActor) error {
	if err := s.segments.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete segment: %w", err)
	}

	auditEvent := &domain.AuditEvent{
		ID:           uuid.New(),
		Action:       domain.AuditActionSegmentDeleted,
		ResourceType: "segment",
		ResourceID:   id,
		ResourceKey:  id.String(),
		Actor:        actor,
		OccurredAt:   time.Now().UTC(),
	}
	_ = s.audit.Create(ctx, auditEvent)

	return nil
}

// toSegmentRulesDomain converts SegmentRuleInput to domain.SegmentRule.
func toSegmentRulesDomain(segmentID uuid.UUID, inputs []SegmentRuleInput) []domain.SegmentRule {
	if len(inputs) == 0 {
		return []domain.SegmentRule{}
	}
	out := make([]domain.SegmentRule, len(inputs))
	for i, sr := range inputs {
		out[i] = domain.SegmentRule{
			SegmentID: segmentID,
			Attribute: sr.Attribute,
			Operator:  sr.Operator,
			Value:     sr.Value,
		}
	}
	return out
}
