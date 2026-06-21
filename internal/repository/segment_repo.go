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

// SegmentRepo handles persistence of segments and their rules.
type SegmentRepo struct {
	db *pgxpool.Pool
}

// NewSegmentRepo creates a new SegmentRepo.
func NewSegmentRepo(db *pgxpool.Pool) *SegmentRepo {
	return &SegmentRepo{db: db}
}

// Create inserts a segment then inserts all segment_rules.
func (r *SegmentRepo) Create(ctx context.Context, seg *domain.Segment) error {
	return WithTx(ctx, r.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO segments (application_id, name, description, operator)
			VALUES ($1, $2, $3, $4)
			RETURNING id, version, created_at, updated_at
		`, seg.ApplicationID, seg.Name, seg.Description, seg.Operator,
		).Scan(&seg.ID, &seg.Version, &seg.CreatedAt, &seg.UpdatedAt)
		if err != nil {
			if strings.Contains(err.Error(), "unique") {
				return domain.ErrAlreadyExists
			}
			return fmt.Errorf("insert segment: %w", err)
		}

		for i := range seg.Rules {
			sr := &seg.Rules[i]
			sr.SegmentID = seg.ID
			if err := tx.QueryRow(ctx, `
				INSERT INTO segment_rules (segment_id, attribute, operator, value)
				VALUES ($1, $2, $3, $4)
				RETURNING id
			`, sr.SegmentID, sr.Attribute, sr.Operator, sr.Value).Scan(&sr.ID); err != nil {
				return fmt.Errorf("insert segment rule: %w", err)
			}
		}
		return nil
	})
}

// GetByID retrieves a segment by UUID. When appID is uuid.Nil the application_id
// constraint is omitted so callers that only have the segment ID can still look up the record.
func (r *SegmentRepo) GetByID(ctx context.Context, appID, segID uuid.UUID) (*domain.Segment, error) {
	seg := &domain.Segment{}
	var err error
	if appID == uuid.Nil {
		err = r.db.QueryRow(ctx, `
			SELECT id, application_id, name, description, operator, version, created_at, updated_at
			FROM segments
			WHERE id = $1
		`, segID).Scan(
			&seg.ID, &seg.ApplicationID, &seg.Name, &seg.Description,
			&seg.Operator, &seg.Version, &seg.CreatedAt, &seg.UpdatedAt,
		)
	} else {
		err = r.db.QueryRow(ctx, `
			SELECT id, application_id, name, description, operator, version, created_at, updated_at
			FROM segments
			WHERE id = $1 AND application_id = $2
		`, segID, appID).Scan(
			&seg.ID, &seg.ApplicationID, &seg.Name, &seg.Description,
			&seg.Operator, &seg.Version, &seg.CreatedAt, &seg.UpdatedAt,
		)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSegmentNotFound
		}
		return nil, fmt.Errorf("get segment by id: %w", err)
	}

	seg.Rules = []domain.SegmentRule{}
	rows, err := r.db.Query(ctx, `
		SELECT id, segment_id, attribute, operator, value
		FROM segment_rules
		WHERE segment_id = $1
	`, seg.ID)
	if err != nil {
		return nil, fmt.Errorf("load segment rules: %w", err)
	}
	for rows.Next() {
		var sr domain.SegmentRule
		if err := rows.Scan(&sr.ID, &sr.SegmentID, &sr.Attribute, &sr.Operator, &sr.Value); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan segment rule: %w", err)
		}
		seg.Rules = append(seg.Rules, sr)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("segment rules rows: %w", err)
	}

	return seg, nil
}

// List returns all segments for an application.
func (r *SegmentRepo) List(ctx context.Context, appID uuid.UUID) ([]*domain.Segment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, application_id, name, description, operator, version, created_at, updated_at
		FROM segments
		WHERE application_id = $1
		ORDER BY created_at DESC
	`, appID)
	if err != nil {
		return nil, fmt.Errorf("list segments: %w", err)
	}
	defer rows.Close()

	var segs []*domain.Segment
	for rows.Next() {
		seg := &domain.Segment{}
		if err := rows.Scan(
			&seg.ID, &seg.ApplicationID, &seg.Name, &seg.Description,
			&seg.Operator, &seg.Version, &seg.CreatedAt, &seg.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan segment: %w", err)
		}
		seg.Rules = []domain.SegmentRule{}
		segs = append(segs, seg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("segments rows: %w", err)
	}
	rows.Close()

	// Load rules for each segment
	for _, seg := range segs {
		ruleRows, err := r.db.Query(ctx, `
			SELECT id, segment_id, attribute, operator, value
			FROM segment_rules
			WHERE segment_id = $1
		`, seg.ID)
		if err != nil {
			return nil, fmt.Errorf("load segment rules for %s: %w", seg.ID, err)
		}
		for ruleRows.Next() {
			var sr domain.SegmentRule
			if err := ruleRows.Scan(&sr.ID, &sr.SegmentID, &sr.Attribute, &sr.Operator, &sr.Value); err != nil {
				ruleRows.Close()
				return nil, fmt.Errorf("scan segment rule: %w", err)
			}
			seg.Rules = append(seg.Rules, sr)
		}
		ruleRows.Close()
		if err := ruleRows.Err(); err != nil {
			return nil, fmt.Errorf("segment rules rows: %w", err)
		}
	}

	return segs, nil
}

// Update replaces a segment's fields and all its rules (DELETE + INSERT).
func (r *SegmentRepo) Update(ctx context.Context, seg *domain.Segment) error {
	return WithTx(ctx, r.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			UPDATE segments
			SET name = $1, description = $2, operator = $3,
			    version = version + 1, updated_at = NOW()
			WHERE id = $4 AND application_id = $5
			RETURNING version, updated_at
		`, seg.Name, seg.Description, seg.Operator, seg.ID, seg.ApplicationID,
		).Scan(&seg.Version, &seg.UpdatedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrSegmentNotFound
			}
			return fmt.Errorf("update segment: %w", err)
		}

		// Replace all rules
		if _, err := tx.Exec(ctx, `DELETE FROM segment_rules WHERE segment_id = $1`, seg.ID); err != nil {
			return fmt.Errorf("delete segment rules: %w", err)
		}
		for i := range seg.Rules {
			sr := &seg.Rules[i]
			sr.SegmentID = seg.ID
			if err := tx.QueryRow(ctx, `
				INSERT INTO segment_rules (segment_id, attribute, operator, value)
				VALUES ($1, $2, $3, $4)
				RETURNING id
			`, sr.SegmentID, sr.Attribute, sr.Operator, sr.Value).Scan(&sr.ID); err != nil {
				return fmt.Errorf("insert segment rule: %w", err)
			}
		}
		return nil
	})
}

// Delete removes a segment by ID.
func (r *SegmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM segments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete segment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrSegmentNotFound
	}
	return nil
}
