package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ecommerce/feature-management/internal/cache"
	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/ecommerce/feature-management/internal/observability"
	"github.com/ecommerce/feature-management/internal/repository"
	"github.com/google/uuid"
)

// CreateFlagRequest carries the input for creating a new feature flag.
type CreateFlagRequest struct {
	ApplicationID uuid.UUID       `json:"application_id" validate:"required"`
	EnvironmentID uuid.UUID       `json:"environment_id" validate:"required"`
	Key           string          `json:"key"            validate:"required,min=1,max=128"`
	Name          string          `json:"name"           validate:"required,min=1,max=256"`
	Description   string          `json:"description"`
	Type          domain.FlagType `json:"type"           validate:"required,oneof=boolean string number json"`
	DefaultValue  json.RawMessage `json:"default_value"  validate:"required"`
	Tags          []string        `json:"tags"`
}

// UpdateFlagRequest carries the input for updating an existing feature flag.
type UpdateFlagRequest struct {
	Name         string          `json:"name"          validate:"required,min=1,max=256"`
	Description  string          `json:"description"`
	Status       domain.FlagStatus `json:"status"      validate:"required,oneof=active inactive archived"`
	DefaultValue json.RawMessage `json:"default_value" validate:"required"`
	Tags         []string        `json:"tags"`
}

// FlagService manages feature flag CRUD and cache invalidation.
type FlagService struct {
	flagRepo  *repository.FlagRepo
	ruleRepo  *repository.RuleRepo
	auditRepo *repository.AuditRepo
	cache     *cache.TieredCache
	metrics   *observability.Metrics
}

// NewFlagService creates a new FlagService.
func NewFlagService(
	flagRepo *repository.FlagRepo,
	ruleRepo *repository.RuleRepo,
	auditRepo *repository.AuditRepo,
	tc *cache.TieredCache,
	metrics *observability.Metrics,
) *FlagService {
	return &FlagService{
		flagRepo:  flagRepo,
		ruleRepo:  ruleRepo,
		auditRepo: auditRepo,
		cache:     tc,
		metrics:   metrics,
	}
}

// GetByKey retrieves a flag by application, environment, and key.
func (s *FlagService) GetByKey(ctx context.Context, appID, envID uuid.UUID, key string) (*domain.FeatureFlag, error) {
	return s.flagRepo.GetByKey(ctx, appID, envID, key)
}

// GetByID retrieves a flag by its UUID.
func (s *FlagService) GetByID(ctx context.Context, id uuid.UUID) (*domain.FeatureFlag, error) {
	return s.flagRepo.GetByID(ctx, id)
}

// List returns all non-deleted flags for an environment.
func (s *FlagService) List(ctx context.Context, appID, envID uuid.UUID) ([]*domain.FeatureFlag, error) {
	return s.flagRepo.List(ctx, appID, envID)
}

// Create creates a new feature flag and records an audit event.
func (s *FlagService) Create(ctx context.Context, req CreateFlagRequest, actor domain.AuditActor) (*domain.FeatureFlag, error) {
	flag := &domain.FeatureFlag{
		ApplicationID: req.ApplicationID,
		EnvironmentID: req.EnvironmentID,
		Key:           req.Key,
		Name:          req.Name,
		Description:   req.Description,
		Type:          req.Type,
		Status:        domain.FlagStatusInactive,
		DefaultValue:  req.DefaultValue,
		Tags:          req.Tags,
	}
	if flag.Tags == nil {
		flag.Tags = []string{}
	}

	if err := s.flagRepo.Create(ctx, flag); err != nil {
		return nil, fmt.Errorf("creating flag: %w", err)
	}

	afterJSON, _ := json.Marshal(flag)
	envID := flag.EnvironmentID
	_ = s.auditRepo.Create(ctx, &domain.AuditEvent{
		ApplicationID: flag.ApplicationID,
		EnvironmentID: &envID,
		Action:        domain.AuditActionFlagCreated,
		ResourceType:  "flag",
		ResourceID:    flag.ID,
		ResourceKey:   flag.Key,
		Actor:         actor,
		After:         json.RawMessage(afterJSON),
		Metadata:      map[string]string{},
		OccurredAt:    time.Now().UTC(),
	})

	if s.metrics != nil {
		s.metrics.FlagUpdatesTotal.WithLabelValues("create").Inc()
	}
	return flag, nil
}

// Update updates a flag's mutable fields and invalidates the cache.
func (s *FlagService) Update(ctx context.Context, id uuid.UUID, req UpdateFlagRequest, actor domain.AuditActor) (*domain.FeatureFlag, error) {
	flag, err := s.flagRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	beforeJSON, _ := json.Marshal(flag)

	flag.Name = req.Name
	flag.Description = req.Description
	flag.Status = req.Status
	flag.DefaultValue = req.DefaultValue
	flag.Tags = req.Tags

	if err := s.flagRepo.Update(ctx, flag); err != nil {
		return nil, fmt.Errorf("updating flag: %w", err)
	}

	// Invalidate cache.
	s.cache.DeleteFlag(ctx, flag.ApplicationID, flag.EnvironmentID, flag.Key)

	afterJSON, _ := json.Marshal(flag)
	envID := flag.EnvironmentID
	_ = s.auditRepo.Create(ctx, &domain.AuditEvent{
		ApplicationID: flag.ApplicationID,
		EnvironmentID: &envID,
		Action:        domain.AuditActionFlagUpdated,
		ResourceType:  "flag",
		ResourceID:    flag.ID,
		ResourceKey:   flag.Key,
		Actor:         actor,
		Before:        json.RawMessage(beforeJSON),
		After:         json.RawMessage(afterJSON),
		Metadata:      map[string]string{},
		OccurredAt:    time.Now().UTC(),
	})

	if s.metrics != nil {
		s.metrics.FlagUpdatesTotal.WithLabelValues("update").Inc()
	}
	return flag, nil
}

// Delete soft-deletes a flag and invalidates the cache.
func (s *FlagService) Delete(ctx context.Context, id uuid.UUID, actor domain.AuditActor) error {
	flag, err := s.flagRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.flagRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting flag: %w", err)
	}

	s.cache.DeleteFlag(ctx, flag.ApplicationID, flag.EnvironmentID, flag.Key)

	envID := flag.EnvironmentID
	_ = s.auditRepo.Create(ctx, &domain.AuditEvent{
		ApplicationID: flag.ApplicationID,
		EnvironmentID: &envID,
		Action:        domain.AuditActionFlagDeleted,
		ResourceType:  "flag",
		ResourceID:    flag.ID,
		ResourceKey:   flag.Key,
		Actor:         actor,
		Metadata:      map[string]string{},
		OccurredAt:    time.Now().UTC(),
	})

	if s.metrics != nil {
		s.metrics.FlagUpdatesTotal.WithLabelValues("delete").Inc()
	}
	return nil
}
