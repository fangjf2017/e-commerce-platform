package api

import (
	"context"

	"github.com/ecommerce/feature-management/internal/domain"
)

type contextKey string

const (
	// ContextKeyActor is the context key for the authenticated actor.
	ContextKeyActor contextKey = "actor"
	// ContextKeyRequestID is the context key for the request ID.
	ContextKeyRequestID contextKey = "request_id"
)

// ActorFromContext extracts the AuditActor from the context.
// Returns an anonymous actor if none is set.
func ActorFromContext(ctx context.Context) domain.AuditActor {
	v := ctx.Value(ContextKeyActor)
	if v == nil {
		return domain.AuditActor{ID: "anonymous", Type: "unknown"}
	}
	actor, ok := v.(domain.AuditActor)
	if !ok {
		return domain.AuditActor{ID: "anonymous", Type: "unknown"}
	}
	return actor
}

// RequestIDFromContext extracts the request ID from the context.
func RequestIDFromContext(ctx context.Context) string {
	v := ctx.Value(ContextKeyRequestID)
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}
