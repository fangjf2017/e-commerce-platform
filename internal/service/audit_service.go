package service

import (
	"context"

	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/ecommerce/feature-management/internal/repository"
	"github.com/google/uuid"
)

// AuditService provides read access to audit events.
type AuditService struct {
	auditRepo *repository.AuditRepo
}

// NewAuditService creates a new AuditService.
func NewAuditService(auditRepo *repository.AuditRepo) *AuditService {
	return &AuditService{auditRepo: auditRepo}
}

// List returns paginated audit events for an application.
func (s *AuditService) List(ctx context.Context, appID uuid.UUID, limit, offset int) ([]*domain.AuditEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.auditRepo.List(ctx, appID, limit, offset)
}
