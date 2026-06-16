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

// SegmentRepo handles persistence of segments and their rules.
type SegmentRepo struct {
	pool *pgxpool.Pool
}

// NewSegmentRepo creates a new SegmentRepo.
func NewSegmentRepo(pool *pgxpool.Pool) *SegmentRepo {
	return &SegmentRepo{pool: pool}
}

// GetByID retrieves a segment with its rules by UUID (satisfies evaluator.SegmentRepository interface).
// appID is accepted for interface compatibility.
func (r *SegmentRepo) GetByID(ctx context.Context, appID, id uuid.UUID) (*domain.Segment, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, application_id, name, description, operator, version, created_at, updated_at
		FROM segments WHERE id = $1
	`, id)

	s, err := scanSegment(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrSegmentNotFound
		}
		return nil, fmt.Errorf("getting segment by id: %w", err)
	}

	rules, err := r.listRules(ctx, s.ID)
	if err != nil {
		return nil, err
	}
	s.Rules = rules
	return s, nil
}

// List returns all segments for an application (without rules for performance).
func (r *SegmentRepo) List(ctx context.Context, appID uuid.UUID) ([]*domain.Segment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, application_id, name, description, operator, version, created_at, updated_at
		FROM segments WHERE application_id = $1 ORDER BY created_at DESC
	`, appID)
	if err != nil {
		return nil, fmt.Errorf("listing segments: %w", err)
	}
	defer rows.Close()

	var segs []*domain.Segment
	for rows.Next() {
		s, err := scanSegment(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning segment: %w", err)
		}
		segs = append(segs, s)
	}
	return segs, rows.Err()
}

// Create inserts a new segment.
func (r *SegmentRepo) Create(ctx context.Context, s *domain.Segment) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO segments (application_id, name, description, operator)
		VALUES ($1, $2, $3, $4)
		RETURNING id, version, created_at, updated_at
	`, s.ApplicationID, s.Name, s.Description, string(s.Operator),
	).Scan(&s.ID, &s.Version, &s.CreatedAt, &s.UpdatedAt)
}

// Update updates a segment's mutable fields.
func (r *SegmentRepo) Update(ctx context.Context, s *domain.Segment) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE segments
		SET name = $1, description = $2, operator = $3, version = version + 1, updated_at = NOW()
		WHERE id = $4
	`, s.Name, s.Description, string(s.Operator), s.ID)
	return err
}

// Delete removes a segment by ID.
func (r *SegmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM segments WHERE id = $1`, id)
	return err
}

// CreateRule inserts a rule for a segment.
func (r *SegmentRepo) CreateRule(ctx context.Context, sr *domain.SegmentRule) error {
	val, err := json.Marshal(sr.Value)
	if err != nil {
		return fmt.Errorf("marshaling segment rule value: %w", err)
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO segment_rules (segment_id, attribute, operator, value)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`, sr.SegmentID, sr.Attribute, string(sr.Operator), val,
	).Scan(&sr.ID, &sr.SegmentID) // placeholder scan
}

// DeleteRulesBySegment removes all rules for a segment.
func (r *SegmentRepo) DeleteRulesBySegment(ctx context.Context, segmentID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM segment_rules WHERE segment_id = $1`, segmentID)
	return err
}

func (r *SegmentRepo) listRules(ctx context.Context, segmentID uuid.UUID) ([]domain.SegmentRule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, segment_id, attribute, operator, value
		FROM segment_rules WHERE segment_id = $1
	`, segmentID)
	if err != nil {
		return nil, fmt.Errorf("listing segment rules: %w", err)
	}
	defer rows.Close()

	var rules []domain.SegmentRule
	for rows.Next() {
		var sr domain.SegmentRule
		var opStr string
		var valBytes []byte
		if err := rows.Scan(&sr.ID, &sr.SegmentID, &sr.Attribute, &opStr, &valBytes); err != nil {
			return nil, fmt.Errorf("scanning segment rule: %w", err)
		}
		sr.Operator = domain.ConditionOperator(opStr)
		sr.Value = json.RawMessage(valBytes)
		rules = append(rules, sr)
	}
	return rules, rows.Err()
}

func scanSegment(row scanner) (*domain.Segment, error) {
	s := &domain.Segment{}
	var opStr string
	err := row.Scan(&s.ID, &s.ApplicationID, &s.Name, &s.Description, &opStr, &s.Version, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	s.Operator = domain.SegmentOperator(opStr)
	return s, nil
}
