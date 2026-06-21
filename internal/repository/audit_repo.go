package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditRepo handles persistence of audit events.
type AuditRepo struct {
	pool *pgxpool.Pool
}

// NewAuditRepo creates a new AuditRepo.
func NewAuditRepo(pool *pgxpool.Pool) *AuditRepo {
	return &AuditRepo{pool: pool}
}

// Create inserts an audit event.
func (r *AuditRepo) Create(ctx context.Context, e *domain.AuditEvent) error {
	var beforeJSON, afterJSON []byte
	var err error
	if e.Before != nil {
		beforeJSON, err = json.Marshal(e.Before)
		if err != nil {
			return fmt.Errorf("marshaling before state: %w", err)
		}
	}
	if e.After != nil {
		afterJSON, err = json.Marshal(e.After)
		if err != nil {
			return fmt.Errorf("marshaling after state: %w", err)
		}
	}
	metaJSON, err := json.Marshal(e.Metadata)
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	return r.pool.QueryRow(ctx, `
		INSERT INTO audit_events
		    (application_id, environment_id, action, resource_type, resource_id,
		     resource_key, actor_id, actor_email, actor_type, before_state, after_state, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, occurred_at
	`, e.ApplicationID, e.EnvironmentID, string(e.Action), e.ResourceType, e.ResourceID,
		e.ResourceKey, e.Actor.ID, e.Actor.Email, e.Actor.Type,
		nullableJSON(beforeJSON), nullableJSON(afterJSON), metaJSON,
	).Scan(&e.ID, &e.OccurredAt)
}

// List returns audit events for an application, most recent first.
func (r *AuditRepo) List(ctx context.Context, appID uuid.UUID, limit, offset int) ([]*domain.AuditEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, application_id, environment_id, action, resource_type, resource_id,
		       resource_key, actor_id, actor_email, actor_type, before_state, after_state, metadata, occurred_at
		FROM audit_events
		WHERE application_id = $1
		ORDER BY occurred_at DESC
		LIMIT $2 OFFSET $3
	`, appID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("listing audit events: %w", err)
	}
	defer rows.Close()

	var events []*domain.AuditEvent
	for rows.Next() {
		e := &domain.AuditEvent{}
		var actionStr string
		var beforeBytes, afterBytes, metaBytes []byte
		if err := rows.Scan(
			&e.ID, &e.ApplicationID, &e.EnvironmentID, &actionStr, &e.ResourceType, &e.ResourceID,
			&e.ResourceKey, &e.Actor.ID, &e.Actor.Email, &e.Actor.Type,
			&beforeBytes, &afterBytes, &metaBytes, &e.OccurredAt,
		); err != nil {
			return nil, fmt.Errorf("scanning audit event: %w", err)
		}
		e.Action = domain.AuditAction(actionStr)
		if beforeBytes != nil {
			e.Before = json.RawMessage(beforeBytes)
		}
		if afterBytes != nil {
			e.After = json.RawMessage(afterBytes)
		}
		if metaBytes != nil {
			_ = json.Unmarshal(metaBytes, &e.Metadata)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// nullableJSON returns nil when the byte slice is empty, otherwise returns it as-is.
// This ensures NULL is stored in Postgres instead of an empty byte slice.
func nullableJSON(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}
