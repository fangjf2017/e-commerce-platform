package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ListAuditFilter provides optional filters for listing audit events.
type ListAuditFilter struct {
	ResourceType *string
	Action       *domain.AuditAction
	ActorID      *string
	After        *time.Time
	Before       *time.Time
	Limit        int
	Offset       int
}

// AuditRepo handles persistence of audit events.
type AuditRepo struct {
	db *pgxpool.Pool
}

// NewAuditRepo creates a new AuditRepo.
func NewAuditRepo(db *pgxpool.Pool) *AuditRepo {
	return &AuditRepo{db: db}
}

// Create inserts a new audit event.
func (r *AuditRepo) Create(ctx context.Context, event *domain.AuditEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}

	err := r.db.QueryRow(ctx, `
		INSERT INTO audit_events
		    (id, application_id, environment_id, action, resource_type, resource_id,
		     resource_key, actor_id, actor_email, actor_type, before_state, after_state,
		     metadata, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING occurred_at
	`,
		event.ID, event.ApplicationID, event.EnvironmentID, event.Action,
		event.ResourceType, event.ResourceID, event.ResourceKey,
		event.Actor.ID, event.Actor.Email, event.Actor.Type,
		event.Before, event.After, event.Metadata, event.OccurredAt,
	).Scan(&event.OccurredAt)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

// List returns paginated audit events for an application with optional filtering.
func (r *AuditRepo) List(ctx context.Context, appID uuid.UUID, filter ListAuditFilter) ([]*domain.AuditEvent, int, error) {
	args := []interface{}{appID}
	where := []string{"application_id = $1"}
	idx := 2

	if filter.ResourceType != nil {
		where = append(where, fmt.Sprintf("resource_type = $%d", idx))
		args = append(args, *filter.ResourceType)
		idx++
	}
	if filter.Action != nil {
		where = append(where, fmt.Sprintf("action = $%d", idx))
		args = append(args, *filter.Action)
		idx++
	}
	if filter.ActorID != nil {
		where = append(where, fmt.Sprintf("actor_id = $%d", idx))
		args = append(args, *filter.ActorID)
		idx++
	}
	if filter.After != nil {
		where = append(where, fmt.Sprintf("occurred_at > $%d", idx))
		args = append(args, *filter.After)
		idx++
	}
	if filter.Before != nil {
		where = append(where, fmt.Sprintf("occurred_at < $%d", idx))
		args = append(args, *filter.Before)
		idx++
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	whereClause := strings.Join(where, " AND ")
	args = append(args, limit, offset)

	query := fmt.Sprintf(`
		SELECT id, application_id, environment_id, action, resource_type, resource_id,
		       resource_key, actor_id, actor_email, actor_type, before_state, after_state,
		       metadata, occurred_at, COUNT(*) OVER() AS total_count
		FROM audit_events
		WHERE %s
		ORDER BY occurred_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, idx, idx+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()

	var events []*domain.AuditEvent
	var total int
	for rows.Next() {
		e := &domain.AuditEvent{}
		if err := rows.Scan(
			&e.ID, &e.ApplicationID, &e.EnvironmentID, &e.Action,
			&e.ResourceType, &e.ResourceID, &e.ResourceKey,
			&e.Actor.ID, &e.Actor.Email, &e.Actor.Type,
			&e.Before, &e.After, &e.Metadata, &e.OccurredAt, &total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan audit event: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("audit events rows: %w", err)
	}
	return events, total, nil
}

// GetByID returns an audit event by ID.
func (r *AuditRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.AuditEvent, error) {
	e := &domain.AuditEvent{}
	err := r.db.QueryRow(ctx, `
		SELECT id, application_id, environment_id, action, resource_type, resource_id,
		       resource_key, actor_id, actor_email, actor_type, before_state, after_state,
		       metadata, occurred_at
		FROM audit_events
		WHERE id = $1
	`, id).Scan(
		&e.ID, &e.ApplicationID, &e.EnvironmentID, &e.Action,
		&e.ResourceType, &e.ResourceID, &e.ResourceKey,
		&e.Actor.ID, &e.Actor.Email, &e.Actor.Type,
		&e.Before, &e.After, &e.Metadata, &e.OccurredAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get audit event by id: %w", err)
	}
	return e, nil
}
