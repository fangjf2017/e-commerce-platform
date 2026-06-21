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

// RuleRepo handles persistence of rules, conditions, and variant allocations.
type RuleRepo struct {
	db *pgxpool.Pool
}

// NewRuleRepo creates a new RuleRepo.
func NewRuleRepo(db *pgxpool.Pool) *RuleRepo {
	return &RuleRepo{db: db}
}

// Create inserts a rule, its conditions, and variant allocations in a tx.
func (r *RuleRepo) Create(ctx context.Context, rule *domain.Rule) error {
	return WithTx(ctx, r.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO rules
			    (flag_id, type, priority, name, description, segment_id, rollout_pct,
			     rollout_salt, schedule_start, schedule_end, enabled)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING id, created_at, updated_at
		`,
			rule.FlagID, rule.Type, rule.Priority, rule.Name, rule.Description,
			rule.SegmentID, rule.RolloutPct, rule.RolloutSalt,
			rule.ScheduleStart, rule.ScheduleEnd, rule.Enabled,
		).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
		if err != nil {
			if strings.Contains(err.Error(), "unique") {
				return domain.ErrAlreadyExists
			}
			return fmt.Errorf("insert rule: %w", err)
		}

		for i := range rule.Conditions {
			c := &rule.Conditions[i]
			c.RuleID = rule.ID
			err := tx.QueryRow(ctx, `
				INSERT INTO conditions (rule_id, attribute, operator, value, negate)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id
			`, c.RuleID, c.Attribute, c.Operator, c.Value, c.Negate).Scan(&c.ID)
			if err != nil {
				return fmt.Errorf("insert condition: %w", err)
			}
		}

		for i := range rule.Variants {
			va := &rule.Variants[i]
			va.RuleID = rule.ID
			err := tx.QueryRow(ctx, `
				INSERT INTO variant_allocations (rule_id, variant_id, rollout_from, rollout_to)
				VALUES ($1, $2, $3, $4)
				RETURNING id
			`, va.RuleID, va.VariantID, va.RolloutFrom, va.RolloutTo).Scan(&va.ID)
			if err != nil {
				return fmt.Errorf("insert variant allocation: %w", err)
			}
		}
		return nil
	})
}

// GetByID returns a rule with conditions and variant allocations.
func (r *RuleRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Rule, error) {
	rule := &domain.Rule{}
	err := r.db.QueryRow(ctx, `
		SELECT id, flag_id, type, priority, name, description,
		       segment_id, rollout_pct, rollout_salt,
		       schedule_start, schedule_end, enabled, created_at, updated_at
		FROM rules
		WHERE id = $1
	`, id).Scan(
		&rule.ID, &rule.FlagID, &rule.Type, &rule.Priority, &rule.Name, &rule.Description,
		&rule.SegmentID, &rule.RolloutPct, &rule.RolloutSalt,
		&rule.ScheduleStart, &rule.ScheduleEnd, &rule.Enabled,
		&rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRuleNotFound
		}
		return nil, fmt.Errorf("get rule by id: %w", err)
	}

	rule.Conditions = []domain.Condition{}
	condRows, err := r.db.Query(ctx, `
		SELECT id, rule_id, attribute, operator, value, negate
		FROM conditions
		WHERE rule_id = $1
	`, rule.ID)
	if err != nil {
		return nil, fmt.Errorf("load conditions: %w", err)
	}
	for condRows.Next() {
		var c domain.Condition
		if err := condRows.Scan(&c.ID, &c.RuleID, &c.Attribute, &c.Operator, &c.Value, &c.Negate); err != nil {
			condRows.Close()
			return nil, fmt.Errorf("scan condition: %w", err)
		}
		rule.Conditions = append(rule.Conditions, c)
	}
	condRows.Close()
	if err := condRows.Err(); err != nil {
		return nil, fmt.Errorf("conditions rows: %w", err)
	}

	rule.Variants = []domain.VariantAllocation{}
	allocRows, err := r.db.Query(ctx, `
		SELECT id, rule_id, variant_id, rollout_from, rollout_to
		FROM variant_allocations
		WHERE rule_id = $1
		ORDER BY rollout_from
	`, rule.ID)
	if err != nil {
		return nil, fmt.Errorf("load variant allocations: %w", err)
	}
	for allocRows.Next() {
		var va domain.VariantAllocation
		if err := allocRows.Scan(&va.ID, &va.RuleID, &va.VariantID, &va.RolloutFrom, &va.RolloutTo); err != nil {
			allocRows.Close()
			return nil, fmt.Errorf("scan variant allocation: %w", err)
		}
		rule.Variants = append(rule.Variants, va)
	}
	allocRows.Close()
	if err := allocRows.Err(); err != nil {
		return nil, fmt.Errorf("variant allocations rows: %w", err)
	}

	return rule, nil
}

// ListByFlag returns all rules for a flag ordered by priority.
func (r *RuleRepo) ListByFlag(ctx context.Context, flagID uuid.UUID) ([]domain.Rule, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, flag_id, type, priority, name, description,
		       segment_id, rollout_pct, rollout_salt,
		       schedule_start, schedule_end, enabled, created_at, updated_at
		FROM rules
		WHERE flag_id = $1
		ORDER BY priority ASC
	`, flagID)
	if err != nil {
		return nil, fmt.Errorf("list rules by flag: %w", err)
	}

	var rules []domain.Rule
	for rows.Next() {
		var rule domain.Rule
		if err := rows.Scan(
			&rule.ID, &rule.FlagID, &rule.Type, &rule.Priority, &rule.Name, &rule.Description,
			&rule.SegmentID, &rule.RolloutPct, &rule.RolloutSalt,
			&rule.ScheduleStart, &rule.ScheduleEnd, &rule.Enabled,
			&rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan rule: %w", err)
		}
		rule.Conditions = []domain.Condition{}
		rule.Variants = []domain.VariantAllocation{}
		rules = append(rules, rule)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rules rows: %w", err)
	}

	if len(rules) == 0 {
		return []domain.Rule{}, nil
	}

	ruleIDs := make([]uuid.UUID, len(rules))
	ruleIndex := make(map[uuid.UUID]int, len(rules))
	for i, rule := range rules {
		ruleIDs[i] = rule.ID
		ruleIndex[rule.ID] = i
	}

	condRows, err := r.db.Query(ctx, `
		SELECT id, rule_id, attribute, operator, value, negate
		FROM conditions
		WHERE rule_id = ANY($1)
		ORDER BY rule_id
	`, ruleIDs)
	if err != nil {
		return nil, fmt.Errorf("load conditions: %w", err)
	}
	for condRows.Next() {
		var c domain.Condition
		if err := condRows.Scan(&c.ID, &c.RuleID, &c.Attribute, &c.Operator, &c.Value, &c.Negate); err != nil {
			condRows.Close()
			return nil, fmt.Errorf("scan condition: %w", err)
		}
		if idx, ok := ruleIndex[c.RuleID]; ok {
			rules[idx].Conditions = append(rules[idx].Conditions, c)
		}
	}
	condRows.Close()
	if err := condRows.Err(); err != nil {
		return nil, fmt.Errorf("conditions rows: %w", err)
	}

	allocRows, err := r.db.Query(ctx, `
		SELECT id, rule_id, variant_id, rollout_from, rollout_to
		FROM variant_allocations
		WHERE rule_id = ANY($1)
		ORDER BY rule_id, rollout_from
	`, ruleIDs)
	if err != nil {
		return nil, fmt.Errorf("load variant allocations: %w", err)
	}
	for allocRows.Next() {
		var va domain.VariantAllocation
		if err := allocRows.Scan(&va.ID, &va.RuleID, &va.VariantID, &va.RolloutFrom, &va.RolloutTo); err != nil {
			allocRows.Close()
			return nil, fmt.Errorf("scan variant allocation: %w", err)
		}
		if idx, ok := ruleIndex[va.RuleID]; ok {
			rules[idx].Variants = append(rules[idx].Variants, va)
		}
	}
	allocRows.Close()
	if err := allocRows.Err(); err != nil {
		return nil, fmt.Errorf("variant allocations rows: %w", err)
	}

	return rules, nil
}

// Update replaces a rule's fields, conditions, and variant allocations.
func (r *RuleRepo) Update(ctx context.Context, rule *domain.Rule) error {
	return WithTx(ctx, r.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			UPDATE rules
			SET type = $1, priority = $2, name = $3, description = $4,
			    segment_id = $5, rollout_pct = $6, rollout_salt = $7,
			    schedule_start = $8, schedule_end = $9, enabled = $10, updated_at = NOW()
			WHERE id = $11
			RETURNING updated_at
		`,
			rule.Type, rule.Priority, rule.Name, rule.Description,
			rule.SegmentID, rule.RolloutPct, rule.RolloutSalt,
			rule.ScheduleStart, rule.ScheduleEnd, rule.Enabled, rule.ID,
		).Scan(&rule.UpdatedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrRuleNotFound
			}
			return fmt.Errorf("update rule: %w", err)
		}

		// Replace conditions
		if _, err := tx.Exec(ctx, `DELETE FROM conditions WHERE rule_id = $1`, rule.ID); err != nil {
			return fmt.Errorf("delete conditions: %w", err)
		}
		for i := range rule.Conditions {
			c := &rule.Conditions[i]
			c.RuleID = rule.ID
			if err := tx.QueryRow(ctx, `
				INSERT INTO conditions (rule_id, attribute, operator, value, negate)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id
			`, c.RuleID, c.Attribute, c.Operator, c.Value, c.Negate).Scan(&c.ID); err != nil {
				return fmt.Errorf("insert condition: %w", err)
			}
		}

		// Replace variant allocations
		if _, err := tx.Exec(ctx, `DELETE FROM variant_allocations WHERE rule_id = $1`, rule.ID); err != nil {
			return fmt.Errorf("delete variant allocations: %w", err)
		}
		for i := range rule.Variants {
			va := &rule.Variants[i]
			va.RuleID = rule.ID
			if err := tx.QueryRow(ctx, `
				INSERT INTO variant_allocations (rule_id, variant_id, rollout_from, rollout_to)
				VALUES ($1, $2, $3, $4)
				RETURNING id
			`, va.RuleID, va.VariantID, va.RolloutFrom, va.RolloutTo).Scan(&va.ID); err != nil {
				return fmt.Errorf("insert variant allocation: %w", err)
			}
		}
		return nil
	})
}

// Delete deletes a rule (cascades to conditions + variant_allocations).
func (r *RuleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM rules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrRuleNotFound
	}
	return nil
}

// ReorderByPriority updates each rule's priority to match the position in ruleIDs slice.
func (r *RuleRepo) ReorderByPriority(ctx context.Context, flagID uuid.UUID, ruleIDs []uuid.UUID) error {
	return WithTx(ctx, r.db, func(tx pgx.Tx) error {
		for priority, ruleID := range ruleIDs {
			tag, err := tx.Exec(ctx, `
				UPDATE rules SET priority = $1, updated_at = NOW()
				WHERE id = $2 AND flag_id = $3
			`, priority, ruleID, flagID)
			if err != nil {
				return fmt.Errorf("update rule priority: %w", err)
			}
			if tag.RowsAffected() == 0 {
				return fmt.Errorf("rule %s not found for flag %s: %w", ruleID, flagID, domain.ErrRuleNotFound)
			}
		}
		return nil
	})
}
