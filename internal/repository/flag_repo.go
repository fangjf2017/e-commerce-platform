package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FlagRepo handles persistence of feature flags and their variants.
type FlagRepo struct {
	pool *pgxpool.Pool
}

// NewFlagRepo creates a new FlagRepo.
func NewFlagRepo(pool *pgxpool.Pool) *FlagRepo {
	return &FlagRepo{pool: pool}
}

// GetByKey retrieves a feature flag (with rules and variants) by app/env/key.
func (r *FlagRepo) GetByKey(ctx context.Context, appID, envID uuid.UUID, key string) (*domain.FeatureFlag, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, application_id, environment_id, key, name, description,
		       type, status, default_value, tags, version, created_at, updated_at
		FROM feature_flags
		WHERE application_id = $1
		  AND environment_id = $2
		  AND key = $3
		  AND deleted_at IS NULL
	`, appID, envID, key)

	f, err := scanFlag(row)
	if err != nil {
		return nil, fmt.Errorf("getting flag by key: %w", err)
	}
	if err := r.loadFlagRelations(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

// GetByID retrieves a feature flag (with rules and variants) by UUID.
func (r *FlagRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.FeatureFlag, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, application_id, environment_id, key, name, description,
		       type, status, default_value, tags, version, created_at, updated_at
		FROM feature_flags
		WHERE id = $1 AND deleted_at IS NULL
	`, id)

	f, err := scanFlag(row)
	if err != nil {
		return nil, fmt.Errorf("getting flag by id: %w", err)
	}
	if err := r.loadFlagRelations(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

// List returns all non-deleted flags for a given environment (without relations).
func (r *FlagRepo) List(ctx context.Context, appID, envID uuid.UUID) ([]*domain.FeatureFlag, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, application_id, environment_id, key, name, description,
		       type, status, default_value, tags, version, created_at, updated_at
		FROM feature_flags
		WHERE application_id = $1
		  AND environment_id = $2
		  AND deleted_at IS NULL
		ORDER BY created_at DESC
	`, appID, envID)
	if err != nil {
		return nil, fmt.Errorf("listing flags: %w", err)
	}
	defer rows.Close()

	var flags []*domain.FeatureFlag
	for rows.Next() {
		f, err := scanFlag(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning flag: %w", err)
		}
		flags = append(flags, f)
	}
	return flags, rows.Err()
}

// Create inserts a new feature flag and returns the created record.
func (r *FlagRepo) Create(ctx context.Context, f *domain.FeatureFlag) error {
	defaultVal, err := json.Marshal(f.DefaultValue)
	if err != nil {
		return fmt.Errorf("marshaling default value: %w", err)
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO feature_flags
		    (application_id, environment_id, key, name, description, type, status, default_value, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, version, created_at, updated_at
	`, f.ApplicationID, f.EnvironmentID, f.Key, f.Name, f.Description,
		string(f.Type), string(f.Status), defaultVal, f.Tags,
	).Scan(&f.ID, &f.Version, &f.CreatedAt, &f.UpdatedAt)
}

// Update updates an existing feature flag's mutable fields.
func (r *FlagRepo) Update(ctx context.Context, f *domain.FeatureFlag) error {
	defaultVal, err := json.Marshal(f.DefaultValue)
	if err != nil {
		return fmt.Errorf("marshaling default value: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE feature_flags
		SET name = $1, description = $2, status = $3, default_value = $4,
		    tags = $5, version = version + 1, updated_at = NOW()
		WHERE id = $6 AND deleted_at IS NULL
	`, f.Name, f.Description, string(f.Status), defaultVal, f.Tags, f.ID)
	return err
}

// Delete soft-deletes a feature flag.
func (r *FlagRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE feature_flags SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	return err
}

// CreateVariant inserts a variant for the given flag.
func (r *FlagRepo) CreateVariant(ctx context.Context, v *domain.Variant) error {
	val, err := json.Marshal(v.Value)
	if err != nil {
		return fmt.Errorf("marshaling variant value: %w", err)
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO variants (flag_id, key, value, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, v.FlagID, v.Key, val, v.Description,
	).Scan(&v.ID)
}

// ListVariants returns all variants for a flag.
func (r *FlagRepo) ListVariants(ctx context.Context, flagID uuid.UUID) ([]domain.Variant, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, flag_id, key, value, description
		FROM variants
		WHERE flag_id = $1
	`, flagID)
	if err != nil {
		return nil, fmt.Errorf("listing variants: %w", err)
	}
	defer rows.Close()

	var variants []domain.Variant
	for rows.Next() {
		var v domain.Variant
		var valBytes []byte
		if err := rows.Scan(&v.ID, &v.FlagID, &v.Key, &valBytes, &v.Description); err != nil {
			return nil, fmt.Errorf("scanning variant: %w", err)
		}
		v.Value = json.RawMessage(valBytes)
		variants = append(variants, v)
	}
	return variants, rows.Err()
}

// loadFlagRelations loads variants and rules for a flag.
func (r *FlagRepo) loadFlagRelations(ctx context.Context, f *domain.FeatureFlag) error {
	variants, err := r.ListVariants(ctx, f.ID)
	if err != nil {
		return err
	}
	f.Variants = variants
	return nil
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanFlag(row scanner) (*domain.FeatureFlag, error) {
	f := &domain.FeatureFlag{}
	var typeStr, statusStr string
	var defaultValBytes []byte
	err := row.Scan(
		&f.ID, &f.ApplicationID, &f.EnvironmentID, &f.Key, &f.Name, &f.Description,
		&typeStr, &statusStr, &defaultValBytes, &f.Tags, &f.Version,
		&f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrFlagNotFound
		}
		return nil, err
	}
	f.Type = domain.FlagType(typeStr)
	f.Status = domain.FlagStatus(statusStr)
	f.DefaultValue = json.RawMessage(defaultValBytes)
	return f, nil
}
