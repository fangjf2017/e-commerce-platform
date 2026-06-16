package repository

import (
	"context"
	"fmt"

	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ApplicationRepo handles persistence of applications.
type ApplicationRepo struct {
	pool *pgxpool.Pool
}

// NewApplicationRepo creates a new ApplicationRepo.
func NewApplicationRepo(pool *pgxpool.Pool) *ApplicationRepo {
	return &ApplicationRepo{pool: pool}
}

// GetByID retrieves an application by UUID.
func (r *ApplicationRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Application, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, name, slug, description, created_at, updated_at
		FROM applications WHERE id = $1 AND deleted_at IS NULL
	`, id)
	a, err := scanApplication(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrApplicationNotFound
		}
		return nil, fmt.Errorf("getting application by id: %w", err)
	}
	return a, nil
}

// GetBySlug retrieves an application by slug.
func (r *ApplicationRepo) GetBySlug(ctx context.Context, slug string) (*domain.Application, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, name, slug, description, created_at, updated_at
		FROM applications WHERE slug = $1 AND deleted_at IS NULL
	`, slug)
	a, err := scanApplication(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrApplicationNotFound
		}
		return nil, fmt.Errorf("getting application by slug: %w", err)
	}
	return a, nil
}

// List returns all non-deleted applications.
func (r *ApplicationRepo) List(ctx context.Context) ([]*domain.Application, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, slug, description, created_at, updated_at
		FROM applications WHERE deleted_at IS NULL ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("listing applications: %w", err)
	}
	defer rows.Close()

	var apps []*domain.Application
	for rows.Next() {
		a, err := scanApplication(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning application: %w", err)
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

// Create inserts a new application.
func (r *ApplicationRepo) Create(ctx context.Context, a *domain.Application) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO applications (name, slug, description)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`, a.Name, a.Slug, a.Description,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

// Update updates an application's mutable fields.
func (r *ApplicationRepo) Update(ctx context.Context, a *domain.Application) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE applications
		SET name = $1, description = $2, updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL
	`, a.Name, a.Description, a.ID)
	return err
}

// Delete soft-deletes an application.
func (r *ApplicationRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE applications SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	return err
}

// EnvironmentRepo handles persistence of environments.
type EnvironmentRepo struct {
	pool *pgxpool.Pool
}

// NewEnvironmentRepo creates a new EnvironmentRepo.
func NewEnvironmentRepo(pool *pgxpool.Pool) *EnvironmentRepo {
	return &EnvironmentRepo{pool: pool}
}

// GetByID retrieves an environment by UUID.
func (r *EnvironmentRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Environment, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, application_id, name, slug, requires_approval, created_at, updated_at
		FROM environments WHERE id = $1 AND deleted_at IS NULL
	`, id)
	e, err := scanEnvironment(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrEnvironmentNotFound
		}
		return nil, fmt.Errorf("getting environment by id: %w", err)
	}
	return e, nil
}

// GetBySlug retrieves an environment by application ID and slug.
func (r *EnvironmentRepo) GetBySlug(ctx context.Context, appID uuid.UUID, slug string) (*domain.Environment, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, application_id, name, slug, requires_approval, created_at, updated_at
		FROM environments
		WHERE application_id = $1 AND slug = $2 AND deleted_at IS NULL
	`, appID, slug)
	e, err := scanEnvironment(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrEnvironmentNotFound
		}
		return nil, fmt.Errorf("getting environment by slug: %w", err)
	}
	return e, nil
}

// ListByApp returns all environments for an application.
func (r *EnvironmentRepo) ListByApp(ctx context.Context, appID uuid.UUID) ([]*domain.Environment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, application_id, name, slug, requires_approval, created_at, updated_at
		FROM environments
		WHERE application_id = $1 AND deleted_at IS NULL ORDER BY created_at ASC
	`, appID)
	if err != nil {
		return nil, fmt.Errorf("listing environments: %w", err)
	}
	defer rows.Close()

	var envs []*domain.Environment
	for rows.Next() {
		e, err := scanEnvironment(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning environment: %w", err)
		}
		envs = append(envs, e)
	}
	return envs, rows.Err()
}

// Create inserts a new environment.
func (r *EnvironmentRepo) Create(ctx context.Context, e *domain.Environment) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO environments (application_id, name, slug, requires_approval)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`, e.ApplicationID, e.Name, e.Slug, e.RequiresApproval,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
}

// Delete soft-deletes an environment.
func (r *EnvironmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE environments SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	return err
}

func scanApplication(row scanner) (*domain.Application, error) {
	a := &domain.Application{}
	err := row.Scan(&a.ID, &a.Name, &a.Slug, &a.Description, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

func scanEnvironment(row scanner) (*domain.Environment, error) {
	e := &domain.Environment{}
	err := row.Scan(&e.ID, &e.ApplicationID, &e.Name, &e.Slug, &e.RequiresApproval, &e.CreatedAt, &e.UpdatedAt)
	return e, err
}
