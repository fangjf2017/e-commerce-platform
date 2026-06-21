package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/ecommerce/feature-management/internal/repository"
	"github.com/google/uuid"
)

// CreateSegmentRequest carries the input for creating a segment.
type CreateSegmentRequest struct {
	ApplicationID uuid.UUID             `json:"application_id" validate:"required"`
	Name          string                `json:"name"           validate:"required,min=1,max=256"`
	Description   string                `json:"description"`
	Operator      domain.SegmentOperator `json:"operator"      validate:"required,oneof=all any"`
	Rules         []SegmentRuleRequest  `json:"rules"`
}

// SegmentRuleRequest carries the input for a single segment rule.
type SegmentRuleRequest struct {
	Attribute string                  `json:"attribute" validate:"required"`
	Operator  domain.ConditionOperator `json:"operator"  validate:"required"`
	Value     json.RawMessage         `json:"value"     validate:"required"`
}

// SegmentService manages segment CRUD.
type SegmentService struct {
	segmentRepo *repository.SegmentRepo
	auditRepo   *repository.AuditRepo
}

// NewSegmentService creates a new SegmentService.
func NewSegmentService(segmentRepo *repository.SegmentRepo, auditRepo *repository.AuditRepo) *SegmentService {
	return &SegmentService{segmentRepo: segmentRepo, auditRepo: auditRepo}
}

// GetByID retrieves a segment by UUID.
func (s *SegmentService) GetByID(ctx context.Context, appID, id uuid.UUID) (*domain.Segment, error) {
	return s.segmentRepo.GetByID(ctx, appID, id)
}

// List returns all segments for an application.
func (s *SegmentService) List(ctx context.Context, appID uuid.UUID) ([]*domain.Segment, error) {
	return s.segmentRepo.List(ctx, appID)
}

// Create creates a new segment with its rules.
func (s *SegmentService) Create(ctx context.Context, req CreateSegmentRequest, actor domain.AuditActor) (*domain.Segment, error) {
	seg := &domain.Segment{
		ApplicationID: req.ApplicationID,
		Name:          req.Name,
		Description:   req.Description,
		Operator:      req.Operator,
	}

	if err := s.segmentRepo.Create(ctx, seg); err != nil {
		return nil, fmt.Errorf("creating segment: %w", err)
	}

	for _, r := range req.Rules {
		sr := &domain.SegmentRule{
			SegmentID: seg.ID,
			Attribute: r.Attribute,
			Operator:  r.Operator,
			Value:     r.Value,
		}
		if err := s.segmentRepo.CreateRule(ctx, sr); err != nil {
			return nil, fmt.Errorf("creating segment rule: %w", err)
		}
		seg.Rules = append(seg.Rules, *sr)
	}

	afterJSON, _ := json.Marshal(seg)
	_ = s.auditRepo.Create(ctx, &domain.AuditEvent{
		ApplicationID: seg.ApplicationID,
		Action:        domain.AuditActionSegmentCreated,
		ResourceType:  "segment",
		ResourceID:    seg.ID,
		ResourceKey:   seg.Name,
		Actor:         actor,
		After:         json.RawMessage(afterJSON),
		Metadata:      map[string]string{},
		OccurredAt:    time.Now().UTC(),
	})
	return seg, nil
}

// Update updates a segment's mutable fields and replaces its rules.
func (s *SegmentService) Update(ctx context.Context, appID, id uuid.UUID, req CreateSegmentRequest, actor domain.AuditActor) (*domain.Segment, error) {
	seg, err := s.segmentRepo.GetByID(ctx, appID, id)
	if err != nil {
		return nil, err
	}

	beforeJSON, _ := json.Marshal(seg)

	seg.Name = req.Name
	seg.Description = req.Description
	seg.Operator = req.Operator

	if err := s.segmentRepo.Update(ctx, seg); err != nil {
		return nil, fmt.Errorf("updating segment: %w", err)
	}

	// Replace rules.
	if err := s.segmentRepo.DeleteRulesBySegment(ctx, seg.ID); err != nil {
		return nil, fmt.Errorf("deleting old segment rules: %w", err)
	}
	seg.Rules = nil
	for _, r := range req.Rules {
		sr := &domain.SegmentRule{
			SegmentID: seg.ID,
			Attribute: r.Attribute,
			Operator:  r.Operator,
			Value:     r.Value,
		}
		if err := s.segmentRepo.CreateRule(ctx, sr); err != nil {
			return nil, fmt.Errorf("creating segment rule: %w", err)
		}
		seg.Rules = append(seg.Rules, *sr)
	}

	afterJSON, _ := json.Marshal(seg)
	_ = s.auditRepo.Create(ctx, &domain.AuditEvent{
		ApplicationID: seg.ApplicationID,
		Action:        domain.AuditActionSegmentUpdated,
		ResourceType:  "segment",
		ResourceID:    seg.ID,
		ResourceKey:   seg.Name,
		Actor:         actor,
		Before:        json.RawMessage(beforeJSON),
		After:         json.RawMessage(afterJSON),
		Metadata:      map[string]string{},
		OccurredAt:    time.Now().UTC(),
	})
	return seg, nil
}

// Delete removes a segment.
func (s *SegmentService) Delete(ctx context.Context, appID, id uuid.UUID, actor domain.AuditActor) error {
	seg, err := s.segmentRepo.GetByID(ctx, appID, id)
	if err != nil {
		return err
	}
	if err := s.segmentRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting segment: %w", err)
	}
	_ = s.auditRepo.Create(ctx, &domain.AuditEvent{
		ApplicationID: seg.ApplicationID,
		Action:        domain.AuditActionSegmentDeleted,
		ResourceType:  "segment",
		ResourceID:    seg.ID,
		ResourceKey:   seg.Name,
		Actor:         actor,
		Metadata:      map[string]string{},
		OccurredAt:    time.Now().UTC(),
	})
	return nil
}
