package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RuleRepo handles persistence of rules, conditions, and variant allocations.
type RuleRepo struct {
	pool *pgxpool.Pool
}

// NewRuleRepo creates a new RuleRepo.
func NewRuleRepo(pool *pgxpool.Pool) *RuleRepo {
	return &RuleRepo{pool: pool}
}

// ListByFlag returns all enabled rules for a flag ordered by priority,
// with conditions and variant allocations populated.
func (r *RuleRepo) ListByFlag(ctx context.Context, flagID uuid.UUID) ([]domain.Rule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, flag_id, type, priority, name, description,
		       segment_id, rollout_pct, rollout_salt,
		       schedule_start, schedule_end, enabled, created_at, updated_at
		FROM rules
		WHERE flag_id = $1 AND enabled = TRUE
		ORDER BY priority ASC
	`, flagID)
	if err != nil {
		return nil, fmt.Errorf("listing rules: %w", err)
	}
	defer rows.Close()

	var rules []domain.Rule
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning rule: %w", err)
		}
		rules = append(rules, *rule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load conditions and allocations for each rule.
	for i := range rules {
		conds, err := r.listConditions(ctx, rules[i].ID)
		if err != nil {
			return nil, err
		}
		rules[i].Conditions = conds

		allocs, err := r.listAllocations(ctx, rules[i].ID)
		if err != nil {
			return nil, err
		}
		rules[i].Variants = allocs
	}
	return rules, nil
}

// GetByID returns a single rule by ID with conditions and allocations.
func (r *RuleRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Rule, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, flag_id, type, priority, name, description,
		       segment_id, rollout_pct, rollout_salt,
		       schedule_start, schedule_end, enabled, created_at, updated_at
		FROM rules WHERE id = $1
	`, id)

	rule, err := scanRule(row)
	if err != nil {
		return nil, fmt.Errorf("getting rule by id: %w", err)
	}

	conds, err := r.listConditions(ctx, rule.ID)
	if err != nil {
		return nil, err
	}
	rule.Conditions = conds

	allocs, err := r.listAllocations(ctx, rule.ID)
	if err != nil {
		return nil, err
	}
	rule.Variants = allocs
	return rule, nil
}

// CreateRule inserts a new rule.
func (r *RuleRepo) CreateRule(ctx context.Context, rule *domain.Rule) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO rules (flag_id, type, priority, name, description, segment_id,
		                   rollout_pct, rollout_salt, schedule_start, schedule_end, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`, rule.FlagID, string(rule.Type), rule.Priority, rule.Name, rule.Description,
		rule.SegmentID, rule.RolloutPct, rule.RolloutSalt,
		rule.ScheduleStart, rule.ScheduleEnd, rule.Enabled,
	).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
}

// UpdateRule updates a rule's mutable fields.
func (r *RuleRepo) UpdateRule(ctx context.Context, rule *domain.Rule) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE rules
		SET priority = $1, name = $2, description = $3, segment_id = $4,
		    rollout_pct = $5, schedule_start = $6, schedule_end = $7,
		    enabled = $8, updated_at = NOW()
		WHERE id = $9
	`, rule.Priority, rule.Name, rule.Description, rule.SegmentID,
		rule.RolloutPct, rule.ScheduleStart, rule.ScheduleEnd, rule.Enabled, rule.ID)
	return err
}

// DeleteRule removes a rule by ID (cascades to conditions and allocations).
func (r *RuleRepo) DeleteRule(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM rules WHERE id = $1`, id)
	return err
}

// CreateCondition inserts a condition for a rule.
func (r *RuleRepo) CreateCondition(ctx context.Context, c *domain.Condition) error {
	val, err := json.Marshal(c.Value)
	if err != nil {
		return fmt.Errorf("marshaling condition value: %w", err)
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO conditions (rule_id, attribute, operator, value, negate)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`, c.RuleID, c.Attribute, string(c.Operator), val, c.Negate,
	).Scan(&c.ID, &c.RuleID) // reuse RuleID scan slot as placeholder
}

// DeleteConditionsByRule removes all conditions for a rule.
func (r *RuleRepo) DeleteConditionsByRule(ctx context.Context, ruleID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM conditions WHERE rule_id = $1`, ruleID)
	return err
}

// CreateAllocation inserts a variant allocation.
func (r *RuleRepo) CreateAllocation(ctx context.Context, a *domain.VariantAllocation) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO variant_allocations (rule_id, variant_id, rollout_from, rollout_to)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, a.RuleID, a.VariantID, a.RolloutFrom, a.RolloutTo,
	).Scan(&a.ID)
}

// DeleteAllocationsByRule removes all variant allocations for a rule.
func (r *RuleRepo) DeleteAllocationsByRule(ctx context.Context, ruleID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM variant_allocations WHERE rule_id = $1`, ruleID)
	return err
}

func (r *RuleRepo) listConditions(ctx context.Context, ruleID uuid.UUID) ([]domain.Condition, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, rule_id, attribute, operator, value, negate
		FROM conditions WHERE rule_id = $1
	`, ruleID)
	if err != nil {
		return nil, fmt.Errorf("listing conditions: %w", err)
	}
	defer rows.Close()

	var conds []domain.Condition
	for rows.Next() {
		var c domain.Condition
		var opStr string
		var valBytes []byte
		if err := rows.Scan(&c.ID, &c.RuleID, &c.Attribute, &opStr, &valBytes, &c.Negate); err != nil {
			return nil, fmt.Errorf("scanning condition: %w", err)
		}
		c.Operator = domain.ConditionOperator(opStr)
		c.Value = json.RawMessage(valBytes)
		conds = append(conds, c)
	}
	return conds, rows.Err()
}

func (r *RuleRepo) listAllocations(ctx context.Context, ruleID uuid.UUID) ([]domain.VariantAllocation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, rule_id, variant_id, rollout_from, rollout_to
		FROM variant_allocations WHERE rule_id = $1
	`, ruleID)
	if err != nil {
		return nil, fmt.Errorf("listing allocations: %w", err)
	}
	defer rows.Close()

	var allocs []domain.VariantAllocation
	for rows.Next() {
		var a domain.VariantAllocation
		if err := rows.Scan(&a.ID, &a.RuleID, &a.VariantID, &a.RolloutFrom, &a.RolloutTo); err != nil {
			return nil, fmt.Errorf("scanning allocation: %w", err)
		}
		allocs = append(allocs, a)
	}
	return allocs, rows.Err()
}

func scanRule(row scanner) (*domain.Rule, error) {
	rule := &domain.Rule{}
	var typeStr string
	if err := row.Scan(
		&rule.ID, &rule.FlagID, &typeStr, &rule.Priority, &rule.Name, &rule.Description,
		&rule.SegmentID, &rule.RolloutPct, &rule.RolloutSalt,
		&rule.ScheduleStart, &rule.ScheduleEnd, &rule.Enabled,
		&rule.CreatedAt, &rule.UpdatedAt,
	); err != nil {
		return nil, err
	}
	rule.Type = domain.RuleType(typeStr)
	return rule, nil
}
