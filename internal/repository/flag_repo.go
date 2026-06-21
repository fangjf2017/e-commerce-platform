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

// FlagRepo handles persistence of feature flags and their related entities.
type FlagRepo struct {
	db *pgxpool.Pool
}

// NewFlagRepo creates a new FlagRepo.
func NewFlagRepo(db *pgxpool.Pool) *FlagRepo {
	return &FlagRepo{db: db}
}

// Create inserts a new flag and its variants in a transaction.
func (r *FlagRepo) Create(ctx context.Context, flag *domain.FeatureFlag) error {
	return WithTx(ctx, r.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO feature_flags
			    (application_id, environment_id, key, name, description, type, status, default_value, tags)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id, version, created_at, updated_at
		`,
			flag.ApplicationID, flag.EnvironmentID, flag.Key, flag.Name, flag.Description,
			flag.Type, flag.Status, flag.DefaultValue, flag.Tags,
		).Scan(&flag.ID, &flag.Version, &flag.CreatedAt, &flag.UpdatedAt)
		if err != nil {
			if strings.Contains(err.Error(), "unique") {
				return domain.ErrAlreadyExists
			}
			return fmt.Errorf("insert feature flag: %w", err)
		}

		for i := range flag.Variants {
			v := &flag.Variants[i]
			v.FlagID = flag.ID
			err := tx.QueryRow(ctx, `
				INSERT INTO variants (flag_id, key, value, description)
				VALUES ($1, $2, $3, $4)
				RETURNING id
			`, v.FlagID, v.Key, v.Value, v.Description).Scan(&v.ID)
			if err != nil {
				return fmt.Errorf("insert variant: %w", err)
			}
		}
		return nil
	})
}

// GetByKey fetches a flag with all its rules, conditions, variants, and variant allocations.
func (r *FlagRepo) GetByKey(ctx context.Context, appID, envID uuid.UUID, key string) (*domain.FeatureFlag, error) {
	flag := &domain.FeatureFlag{}
	err := r.db.QueryRow(ctx, `
		SELECT id, application_id, environment_id, key, name, description,
		       type, status, default_value, tags, version, created_at, updated_at
		FROM feature_flags
		WHERE application_id = $1
		  AND environment_id = $2
		  AND key = $3
		  AND deleted_at IS NULL
	`, appID, envID, key).Scan(
		&flag.ID, &flag.ApplicationID, &flag.EnvironmentID, &flag.Key, &flag.Name, &flag.Description,
		&flag.Type, &flag.Status, &flag.DefaultValue, &flag.Tags, &flag.Version,
		&flag.CreatedAt, &flag.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrFlagNotFound
		}
		return nil, fmt.Errorf("get flag by key: %w", err)
	}

	if err := r.loadRelations(ctx, flag); err != nil {
		return nil, err
	}
	return flag, nil
}

// GetByID fetches a flag by ID with all relations.
func (r *FlagRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.FeatureFlag, error) {
	flag := &domain.FeatureFlag{}
	err := r.db.QueryRow(ctx, `
		SELECT id, application_id, environment_id, key, name, description,
		       type, status, default_value, tags, version, created_at, updated_at
		FROM feature_flags
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(
		&flag.ID, &flag.ApplicationID, &flag.EnvironmentID, &flag.Key, &flag.Name, &flag.Description,
		&flag.Type, &flag.Status, &flag.DefaultValue, &flag.Tags, &flag.Version,
		&flag.CreatedAt, &flag.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrFlagNotFound
		}
		return nil, fmt.Errorf("get flag by id: %w", err)
	}

	if err := r.loadRelations(ctx, flag); err != nil {
		return nil, err
	}
	return flag, nil
}

// List returns paginated flags with optional status/tags filter and total count.
func (r *FlagRepo) List(ctx context.Context, appID, envID uuid.UUID, status *domain.FlagStatus, tags []string, limit, offset int) ([]*domain.FeatureFlag, int, error) {
	args := []interface{}{appID, envID}
	where := []string{"application_id = $1", "environment_id = $2", "deleted_at IS NULL"}
	idx := 3

	if status != nil {
		where = append(where, fmt.Sprintf("status = $%d", idx))
		args = append(args, *status)
		idx++
	}
	if len(tags) > 0 {
		where = append(where, fmt.Sprintf("tags && $%d", idx))
		args = append(args, tags)
		idx++
	}

	whereClause := strings.Join(where, " AND ")
	args = append(args, limit, offset)

	query := fmt.Sprintf(`
		SELECT id, application_id, environment_id, key, name, description,
		       type, status, default_value, tags, version, created_at, updated_at,
		       COUNT(*) OVER() AS total_count
		FROM feature_flags
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, idx, idx+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list flags: %w", err)
	}
	defer rows.Close()

	var flags []*domain.FeatureFlag
	var total int
	for rows.Next() {
		f := &domain.FeatureFlag{}
		if err := rows.Scan(
			&f.ID, &f.ApplicationID, &f.EnvironmentID, &f.Key, &f.Name, &f.Description,
			&f.Type, &f.Status, &f.DefaultValue, &f.Tags, &f.Version,
			&f.CreatedAt, &f.UpdatedAt, &total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan flag row: %w", err)
		}
		flags = append(flags, f)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list flags rows: %w", err)
	}
	return flags, total, nil
}

// ListByEnvironment returns all active flags for an environment (used for batch evaluation).
func (r *FlagRepo) ListByEnvironment(ctx context.Context, appID, envID uuid.UUID) ([]*domain.FeatureFlag, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, application_id, environment_id, key, name, description,
		       type, status, default_value, tags, version, created_at, updated_at
		FROM feature_flags
		WHERE application_id = $1
		  AND environment_id = $2
		  AND status = 'active'
		  AND deleted_at IS NULL
		ORDER BY created_at DESC
	`, appID, envID)
	if err != nil {
		return nil, fmt.Errorf("list flags by environment: %w", err)
	}
	defer rows.Close()

	var flags []*domain.FeatureFlag
	for rows.Next() {
		f := &domain.FeatureFlag{}
		if err := rows.Scan(
			&f.ID, &f.ApplicationID, &f.EnvironmentID, &f.Key, &f.Name, &f.Description,
			&f.Type, &f.Status, &f.DefaultValue, &f.Tags, &f.Version,
			&f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan flag row: %w", err)
		}
		flags = append(flags, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list flags by environment rows: %w", err)
	}
	rows.Close()

	for _, f := range flags {
		if err := r.loadRelations(ctx, f); err != nil {
			return nil, err
		}
	}
	return flags, nil
}

// Update updates a flag's mutable fields and increments version.
func (r *FlagRepo) Update(ctx context.Context, flag *domain.FeatureFlag) error {
	err := r.db.QueryRow(ctx, `
		UPDATE feature_flags
		SET name = $1, description = $2, default_value = $3, tags = $4,
		    version = version + 1, updated_at = NOW()
		WHERE id = $5 AND deleted_at IS NULL
		RETURNING version, updated_at
	`, flag.Name, flag.Description, flag.DefaultValue, flag.Tags, flag.ID,
	).Scan(&flag.Version, &flag.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrFlagNotFound
		}
		return fmt.Errorf("update flag: %w", err)
	}
	return nil
}

// Delete soft-deletes a flag.
func (r *FlagRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE feature_flags
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("delete flag: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrFlagNotFound
	}
	return nil
}

// UpdateStatus changes the flag status and increments version.
func (r *FlagRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.FlagStatus) (*domain.FeatureFlag, error) {
	flag := &domain.FeatureFlag{}
	err := r.db.QueryRow(ctx, `
		UPDATE feature_flags
		SET status = $1, version = version + 1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING id, application_id, environment_id, key, name, description,
		          type, status, default_value, tags, version, created_at, updated_at
	`, status, id).Scan(
		&flag.ID, &flag.ApplicationID, &flag.EnvironmentID, &flag.Key, &flag.Name, &flag.Description,
		&flag.Type, &flag.Status, &flag.DefaultValue, &flag.Tags, &flag.Version,
		&flag.CreatedAt, &flag.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrFlagNotFound
		}
		return nil, fmt.Errorf("update flag status: %w", err)
	}
	return flag, nil
}

// loadRelations loads rules (with conditions + variant allocations) and variants for a flag.
func (r *FlagRepo) loadRelations(ctx context.Context, flag *domain.FeatureFlag) error {
	// Load variants
	varRows, err := r.db.Query(ctx, `
		SELECT id, flag_id, key, value, description
		FROM variants
		WHERE flag_id = $1
		ORDER BY key
	`, flag.ID)
	if err != nil {
		return fmt.Errorf("load variants: %w", err)
	}
	flag.Variants = []domain.Variant{}
	for varRows.Next() {
		var v domain.Variant
		if err := varRows.Scan(&v.ID, &v.FlagID, &v.Key, &v.Value, &v.Description); err != nil {
			varRows.Close()
			return fmt.Errorf("scan variant: %w", err)
		}
		flag.Variants = append(flag.Variants, v)
	}
	varRows.Close()
	if err := varRows.Err(); err != nil {
		return fmt.Errorf("variants rows: %w", err)
	}

	// Load rules
	ruleRows, err := r.db.Query(ctx, `
		SELECT id, flag_id, type, priority, name, description,
		       segment_id, rollout_pct, rollout_salt,
		       schedule_start, schedule_end, enabled, created_at, updated_at
		FROM rules
		WHERE flag_id = $1
		ORDER BY priority ASC
	`, flag.ID)
	if err != nil {
		return fmt.Errorf("load rules: %w", err)
	}

	var rules []domain.Rule
	for ruleRows.Next() {
		var rule domain.Rule
		if err := ruleRows.Scan(
			&rule.ID, &rule.FlagID, &rule.Type, &rule.Priority, &rule.Name, &rule.Description,
			&rule.SegmentID, &rule.RolloutPct, &rule.RolloutSalt,
			&rule.ScheduleStart, &rule.ScheduleEnd, &rule.Enabled,
			&rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			ruleRows.Close()
			return fmt.Errorf("scan rule: %w", err)
		}
		rule.Conditions = []domain.Condition{}
		rule.Variants = []domain.VariantAllocation{}
		rules = append(rules, rule)
	}
	ruleRows.Close()
	if err := ruleRows.Err(); err != nil {
		return fmt.Errorf("rules rows: %w", err)
	}

	if len(rules) == 0 {
		flag.Rules = []domain.Rule{}
		return nil
	}

	// Collect rule IDs for batch loading
	ruleIDs := make([]uuid.UUID, len(rules))
	ruleIndex := make(map[uuid.UUID]int, len(rules))
	for i, rule := range rules {
		ruleIDs[i] = rule.ID
		ruleIndex[rule.ID] = i
	}

	// Load conditions for all rules in one query
	condRows, err := r.db.Query(ctx, `
		SELECT id, rule_id, attribute, operator, value, negate
		FROM conditions
		WHERE rule_id = ANY($1)
		ORDER BY rule_id
	`, ruleIDs)
	if err != nil {
		return fmt.Errorf("load conditions: %w", err)
	}
	for condRows.Next() {
		var c domain.Condition
		if err := condRows.Scan(&c.ID, &c.RuleID, &c.Attribute, &c.Operator, &c.Value, &c.Negate); err != nil {
			condRows.Close()
			return fmt.Errorf("scan condition: %w", err)
		}
		if idx, ok := ruleIndex[c.RuleID]; ok {
			rules[idx].Conditions = append(rules[idx].Conditions, c)
		}
	}
	condRows.Close()
	if err := condRows.Err(); err != nil {
		return fmt.Errorf("conditions rows: %w", err)
	}

	// Load variant allocations for all rules in one query
	allocRows, err := r.db.Query(ctx, `
		SELECT id, rule_id, variant_id, rollout_from, rollout_to
		FROM variant_allocations
		WHERE rule_id = ANY($1)
		ORDER BY rule_id, rollout_from
	`, ruleIDs)
	if err != nil {
		return fmt.Errorf("load variant allocations: %w", err)
	}
	for allocRows.Next() {
		var va domain.VariantAllocation
		if err := allocRows.Scan(&va.ID, &va.RuleID, &va.VariantID, &va.RolloutFrom, &va.RolloutTo); err != nil {
			allocRows.Close()
			return fmt.Errorf("scan variant allocation: %w", err)
		}
		if idx, ok := ruleIndex[va.RuleID]; ok {
			rules[idx].Variants = append(rules[idx].Variants, va)
		}
	}
	allocRows.Close()
	if err := allocRows.Err(); err != nil {
		return fmt.Errorf("variant allocations rows: %w", err)
	}

	flag.Rules = rules
	return nil
}
