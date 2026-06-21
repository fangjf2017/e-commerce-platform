package service

import (
	"context"
	"fmt"

	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/ecommerce/feature-management/internal/repository"
	"github.com/google/uuid"
)

// ApplicationService manages applications and their environments.
type ApplicationService struct {
	apps *repository.ApplicationRepo
	envs *repository.EnvironmentRepo
}

// NewApplicationService creates a new ApplicationService.
func NewApplicationService(apps *repository.ApplicationRepo, envs *repository.EnvironmentRepo) *ApplicationService {
	return &ApplicationService{apps: apps, envs: envs}
}

// Create creates a new application.
func (s *ApplicationService) Create(ctx context.Context, inp CreateApplicationInput) (*domain.Application, error) {
	app := &domain.Application{
		Name:        inp.Name,
		Slug:        inp.Slug,
		Description: inp.Description,
	}
	if err := s.apps.Create(ctx, app); err != nil {
		return nil, fmt.Errorf("create application: %w", err)
	}
	return app, nil
}

// Get returns an application by ID.
func (s *ApplicationService) Get(ctx context.Context, id uuid.UUID) (*domain.Application, error) {
	app, err := s.apps.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get application: %w", err)
	}
	return app, nil
}

// GetBySlug returns an application by slug.
func (s *ApplicationService) GetBySlug(ctx context.Context, slug string) (*domain.Application, error) {
	app, err := s.apps.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("get application by slug: %w", err)
	}
	return app, nil
}

// List returns paginated applications with total count.
func (s *ApplicationService) List(ctx context.Context, inp ListApplicationsInput) ([]*domain.Application, int, error) {
	limit := inp.Limit
	if limit <= 0 {
		limit = 20
	}
	apps, total, err := s.apps.List(ctx, limit, inp.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list applications: %w", err)
	}
	return apps, total, nil
}

// Update updates an application's mutable fields.
func (s *ApplicationService) Update(ctx context.Context, id uuid.UUID, inp UpdateApplicationInput) (*domain.Application, error) {
	app, err := s.apps.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get application for update: %w", err)
	}
	app.Name = inp.Name
	app.Description = inp.Description

	if err := s.apps.Update(ctx, app); err != nil {
		return nil, fmt.Errorf("update application: %w", err)
	}
	return app, nil
}

// Delete removes an application by ID.
func (s *ApplicationService) Delete(ctx context.Context, id uuid.UUID, actor domain.AuditActor) error {
	_ = actor
	if err := s.apps.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete application: %w", err)
	}
	return nil
}

// CreateEnvironment creates a new environment for an application.
func (s *ApplicationService) CreateEnvironment(ctx context.Context, inp CreateEnvironmentInput) (*domain.Environment, error) {
	env := &domain.Environment{
		ApplicationID:    inp.ApplicationID,
		Name:             inp.Name,
		Slug:             inp.Slug,
		RequiresApproval: inp.RequiresApproval,
	}
	if err := s.envs.Create(ctx, env); err != nil {
		return nil, fmt.Errorf("create environment: %w", err)
	}
	return env, nil
}

// GetEnvironment returns an environment by env ID only.
func (s *ApplicationService) GetEnvironment(ctx context.Context, envID uuid.UUID) (*domain.Environment, error) {
	env, err := s.envs.GetByID(ctx, uuid.Nil, envID)
	if err != nil {
		return nil, fmt.Errorf("get environment: %w", err)
	}
	return env, nil
}

// ListEnvironments returns environments for an application with pagination.
func (s *ApplicationService) ListEnvironments(ctx context.Context, appID uuid.UUID, limit, offset int) ([]*domain.Environment, int, error) {
	envs, err := s.envs.List(ctx, appID)
	if err != nil {
		return nil, 0, fmt.Errorf("list environments: %w", err)
	}
	total := len(envs)

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= total {
		return []*domain.Environment{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return envs[offset:end], total, nil
}

// UpdateEnvironment updates an environment's mutable fields by envID.
func (s *ApplicationService) UpdateEnvironment(ctx context.Context, envID uuid.UUID, inp UpdateEnvironmentInput) (*domain.Environment, error) {
	env, err := s.envs.GetByID(ctx, uuid.Nil, envID)
	if err != nil {
		return nil, fmt.Errorf("get environment for update: %w", err)
	}
	env.Name = inp.Name
	env.RequiresApproval = inp.RequiresApproval

	if err := s.envs.Update(ctx, env); err != nil {
		return nil, fmt.Errorf("update environment: %w", err)
	}
	return env, nil
}

// DeleteEnvironment removes an environment by ID.
func (s *ApplicationService) DeleteEnvironment(ctx context.Context, envID uuid.UUID, actor domain.AuditActor) error {
	_ = actor
	if err := s.envs.Delete(ctx, envID); err != nil {
		return fmt.Errorf("delete environment: %w", err)
	}
	return nil
}
