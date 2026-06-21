package service

import (
	"context"
	"fmt"

	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/ecommerce/feature-management/internal/repository"
	"github.com/google/uuid"
)

// AuditService provides access to audit event history.
type AuditService struct {
	repo *repository.AuditRepo
}

// NewAuditService creates a new AuditService.
func NewAuditService(repo *repository.AuditRepo) *AuditService {
	return &AuditService{repo: repo}
}

// List returns paginated audit events for an application with optional filters.
// Uses the ListAuditInput type defined in service.go.
func (s *AuditService) List(ctx context.Context, inp ListAuditInput) ([]*domain.AuditEvent, int, error) {
	filter := repository.ListAuditFilter{
		Limit:  inp.Limit,
		Offset: inp.Offset,
		After:  inp.After,
		Before: inp.Before,
	}
	if inp.ResourceType != "" {
		filter.ResourceType = &inp.ResourceType
	}
	if inp.Action != "" {
		action := domain.AuditAction(inp.Action)
		filter.Action = &action
	}
	if inp.ActorID != "" {
		filter.ActorID = &inp.ActorID
	}

	events, total, err := s.repo.List(ctx, inp.ApplicationID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit events: %w", err)
	}
	return events, total, nil
}

// GetEvent returns a single audit event by ID.
func (s *AuditService) GetEvent(ctx context.Context, id uuid.UUID) (*domain.AuditEvent, error) {
	event, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get audit event: %w", err)
	}
	return event, nil
}
