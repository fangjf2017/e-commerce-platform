package service

import (
	"context"
	"fmt"

	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/ecommerce/feature-management/internal/repository"
	"github.com/google/uuid"
)

// CreateApplicationRequest carries input for creating an application.
type CreateApplicationRequest struct {
	Name        string `json:"name"        validate:"required,min=1,max=256"`
	Slug        string `json:"slug"        validate:"required,min=1,max=128,slug"`
	Description string `json:"description"`
}

// CreateEnvironmentRequest carries input for creating an environment.
type CreateEnvironmentRequest struct {
	ApplicationID    uuid.UUID `json:"application_id"    validate:"required"`
	Name             string    `json:"name"              validate:"required,min=1,max=256"`
	Slug             string    `json:"slug"              validate:"required,min=1,max=128"`
	RequiresApproval bool      `json:"requires_approval"`
}

// ApplicationService manages application and environment CRUD.
type ApplicationService struct {
	appRepo *repository.ApplicationRepo
	envRepo *repository.EnvironmentRepo
}

// NewApplicationService creates a new ApplicationService.
func NewApplicationService(appRepo *repository.ApplicationRepo, envRepo *repository.EnvironmentRepo) *ApplicationService {
	return &ApplicationService{appRepo: appRepo, envRepo: envRepo}
}

// GetApplication retrieves an application by UUID.
func (s *ApplicationService) GetApplication(ctx context.Context, id uuid.UUID) (*domain.Application, error) {
	return s.appRepo.GetByID(ctx, id)
}

// GetApplicationBySlug retrieves an application by slug.
func (s *ApplicationService) GetApplicationBySlug(ctx context.Context, slug string) (*domain.Application, error) {
	return s.appRepo.GetBySlug(ctx, slug)
}

// ListApplications returns all applications.
func (s *ApplicationService) ListApplications(ctx context.Context) ([]*domain.Application, error) {
	return s.appRepo.List(ctx)
}

// CreateApplication creates a new application.
func (s *ApplicationService) CreateApplication(ctx context.Context, req CreateApplicationRequest) (*domain.Application, error) {
	app := &domain.Application{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	}
	if err := s.appRepo.Create(ctx, app); err != nil {
		return nil, fmt.Errorf("creating application: %w", err)
	}
	return app, nil
}

// UpdateApplication updates an application's mutable fields.
func (s *ApplicationService) UpdateApplication(ctx context.Context, id uuid.UUID, req CreateApplicationRequest) (*domain.Application, error) {
	app, err := s.appRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	app.Name = req.Name
	app.Description = req.Description
	if err := s.appRepo.Update(ctx, app); err != nil {
		return nil, fmt.Errorf("updating application: %w", err)
	}
	return app, nil
}

// DeleteApplication soft-deletes an application.
func (s *ApplicationService) DeleteApplication(ctx context.Context, id uuid.UUID) error {
	return s.appRepo.Delete(ctx, id)
}

// GetEnvironment retrieves an environment by UUID.
func (s *ApplicationService) GetEnvironment(ctx context.Context, appID, id uuid.UUID) (*domain.Environment, error) {
	return s.envRepo.GetByID(ctx, appID, id)
}

// ListEnvironments returns all environments for an application.
func (s *ApplicationService) ListEnvironments(ctx context.Context, appID uuid.UUID) ([]*domain.Environment, error) {
	return s.envRepo.ListByApp(ctx, appID)
}

// CreateEnvironment creates a new environment.
func (s *ApplicationService) CreateEnvironment(ctx context.Context, req CreateEnvironmentRequest) (*domain.Environment, error) {
	env := &domain.Environment{
		ApplicationID:    req.ApplicationID,
		Name:             req.Name,
		Slug:             req.Slug,
		RequiresApproval: req.RequiresApproval,
	}
	if err := s.envRepo.Create(ctx, env); err != nil {
		return nil, fmt.Errorf("creating environment: %w", err)
	}
	return env, nil
}

// DeleteEnvironment soft-deletes an environment.
func (s *ApplicationService) DeleteEnvironment(ctx context.Context, id uuid.UUID) error {
	return s.envRepo.Delete(ctx, id)
}
