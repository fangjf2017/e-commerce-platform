package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ApplicationRepo handles persistence of applications.
type ApplicationRepo struct {
	db *pgxpool.Pool
}

// NewApplicationRepo creates a new ApplicationRepo.
func NewApplicationRepo(db *pgxpool.Pool) *ApplicationRepo {
	return &ApplicationRepo{db: db}
}

// Create inserts a new application.
func (r *ApplicationRepo) Create(ctx context.Context, app *domain.Application) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO applications (name, slug, description)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`, app.Name, app.Slug, app.Description,
	).Scan(&app.ID, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("insert application: %w", err)
	}
	return nil
}

// GetByID retrieves an application by UUID.
func (r *ApplicationRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Application, error) {
	app := &domain.Application{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name, slug, description, created_at, updated_at
		FROM applications
		WHERE id = $1
	`, id).Scan(&app.ID, &app.Name, &app.Slug, &app.Description, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrApplicationNotFound
		}
		return nil, fmt.Errorf("get application by id: %w", err)
	}
	return app, nil
}

// GetBySlug retrieves an application by slug.
func (r *ApplicationRepo) GetBySlug(ctx context.Context, slug string) (*domain.Application, error) {
	app := &domain.Application{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name, slug, description, created_at, updated_at
		FROM applications
		WHERE slug = $1
	`, slug).Scan(&app.ID, &app.Name, &app.Slug, &app.Description, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrApplicationNotFound
		}
		return nil, fmt.Errorf("get application by slug: %w", err)
	}
	return app, nil
}

// List returns paginated applications with total count.
func (r *ApplicationRepo) List(ctx context.Context, limit, offset int) ([]*domain.Application, int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, slug, description, created_at, updated_at,
		       COUNT(*) OVER() AS total_count
		FROM applications
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list applications: %w", err)
	}
	defer rows.Close()

	var apps []*domain.Application
	var total int
	for rows.Next() {
		app := &domain.Application{}
		if err := rows.Scan(
			&app.ID, &app.Name, &app.Slug, &app.Description,
			&app.CreatedAt, &app.UpdatedAt, &total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan application: %w", err)
		}
		apps = append(apps, app)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("applications rows: %w", err)
	}
	return apps, total, nil
}

// Update updates an application's mutable fields.
func (r *ApplicationRepo) Update(ctx context.Context, app *domain.Application) error {
	err := r.db.QueryRow(ctx, `
		UPDATE applications
		SET name = $1, description = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING updated_at
	`, app.Name, app.Description, app.ID).Scan(&app.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrApplicationNotFound
		}
		return fmt.Errorf("update application: %w", err)
	}
	return nil
}

// Delete removes an application by ID.
func (r *ApplicationRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM applications WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete application: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrApplicationNotFound
	}
	return nil
}

// EnvironmentRepo handles persistence of environments.
type EnvironmentRepo struct {
	db *pgxpool.Pool
}

// NewEnvironmentRepo creates a new EnvironmentRepo.
func NewEnvironmentRepo(db *pgxpool.Pool) *EnvironmentRepo {
	return &EnvironmentRepo{db: db}
}

// Create inserts a new environment.
func (r *EnvironmentRepo) Create(ctx context.Context, env *domain.Environment) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO environments (application_id, name, slug, requires_approval)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`, env.ApplicationID, env.Name, env.Slug, env.RequiresApproval,
	).Scan(&env.ID, &env.CreatedAt, &env.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("insert environment: %w", err)
	}
	return nil
}

// GetByIDOnly retrieves an environment by UUID without requiring appID.
func (r *EnvironmentRepo) GetByIDOnly(ctx context.Context, envID uuid.UUID) (*domain.Environment, error) {
	env := &domain.Environment{}
	err := r.db.QueryRow(ctx, `
		SELECT id, application_id, name, slug, requires_approval, created_at, updated_at
		FROM environments WHERE id = $1
	`, envID).Scan(
		&env.ID, &env.ApplicationID, &env.Name, &env.Slug,
		&env.RequiresApproval, &env.CreatedAt, &env.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrEnvironmentNotFound
		}
		return nil, fmt.Errorf("get environment by id: %w", err)
	}
	return env, nil
}

// GetByID retrieves an environment by UUID (scoped to appID).
func (r *EnvironmentRepo) GetByID(ctx context.Context, appID, envID uuid.UUID) (*domain.Environment, error) {
	env := &domain.Environment{}
	err := r.db.QueryRow(ctx, `
		SELECT id, application_id, name, slug, requires_approval, created_at, updated_at
		FROM environments
		WHERE id = $1 AND application_id = $2
	`, envID, appID).Scan(
		&env.ID, &env.ApplicationID, &env.Name, &env.Slug,
		&env.RequiresApproval, &env.CreatedAt, &env.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrEnvironmentNotFound
		}
		return nil, fmt.Errorf("get environment by id: %w", err)
	}
	return env, nil
}

// List returns all environments for an application.
func (r *EnvironmentRepo) List(ctx context.Context, appID uuid.UUID) ([]*domain.Environment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, application_id, name, slug, requires_approval, created_at, updated_at
		FROM environments
		WHERE application_id = $1
		ORDER BY created_at ASC
	`, appID)
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}
	defer rows.Close()

	var envs []*domain.Environment
	for rows.Next() {
		env := &domain.Environment{}
		if err := rows.Scan(
			&env.ID, &env.ApplicationID, &env.Name, &env.Slug,
			&env.RequiresApproval, &env.CreatedAt, &env.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan environment: %w", err)
		}
		envs = append(envs, env)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("environments rows: %w", err)
	}
	return envs, nil
}

// Update updates an environment's mutable fields.
func (r *EnvironmentRepo) Update(ctx context.Context, env *domain.Environment) error {
	err := r.db.QueryRow(ctx, `
		UPDATE environments
		SET name = $1, requires_approval = $2, updated_at = NOW()
		WHERE id = $3 AND application_id = $4
		RETURNING updated_at
	`, env.Name, env.RequiresApproval, env.ID, env.ApplicationID).Scan(&env.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrEnvironmentNotFound
		}
		return fmt.Errorf("update environment: %w", err)
	}
	return nil
}

// Delete removes an environment by ID.
func (r *EnvironmentRepo) Delete(ctx context.Context, envID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM environments WHERE id = $1`, envID)
	if err != nil {
		return fmt.Errorf("delete environment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrEnvironmentNotFound
	}
	return nil
}
