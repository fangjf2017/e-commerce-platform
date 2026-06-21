// Package service contains the business logic layer for the Feature Management Service.
// Input types for each service are defined alongside the service that uses them.
package service

import (
	"encoding/json"
	"time"

	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/google/uuid"
)

// ─── Application inputs ───────────────────────────────────────────────────────

// CreateApplicationInput holds the data needed to create an application.
type CreateApplicationInput struct {
	Name        string
	Slug        string
	Description string
	Actor       domain.AuditActor
}

// UpdateApplicationInput holds the data needed to update an application.
type UpdateApplicationInput struct {
	Name        string
	Description string
	Actor       domain.AuditActor
}

// ListApplicationsInput holds pagination parameters for listing applications.
type ListApplicationsInput struct {
	Limit  int
	Offset int
}

// ─── Environment inputs ───────────────────────────────────────────────────────

// CreateEnvironmentInput holds the data needed to create an environment.
type CreateEnvironmentInput struct {
	ApplicationID    uuid.UUID
	Name             string
	Slug             string
	RequiresApproval bool
	Actor            domain.AuditActor
}

// UpdateEnvironmentInput holds the data needed to update an environment.
type UpdateEnvironmentInput struct {
	Name             string
	RequiresApproval bool
	Actor            domain.AuditActor
}

// ─── Audit inputs ─────────────────────────────────────────────────────────────

// ListAuditInput holds filter and pagination parameters for listing audit events.
type ListAuditInput struct {
	ApplicationID uuid.UUID
	ResourceType  string
	Action        string
	ActorID       string
	After         *time.Time
	Before        *time.Time
	Limit         int
	Offset        int
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// conditionInputsToDomain converts ConditionInput slices to domain.Condition slices.
func conditionInputsToDomain(inputs []ConditionInput) []domain.Condition {
	out := make([]domain.Condition, len(inputs))
	for i, c := range inputs {
		out[i] = domain.Condition{
			Attribute: c.Attribute,
			Operator:  c.Operator,
			Value:     json.RawMessage(c.Value),
			Negate:    c.Negate,
		}
	}
	return out
}

// segmentRuleInputsToDomain converts SegmentRuleInput slices to domain.SegmentRule slices.
func segmentRuleInputsToDomain(segmentID uuid.UUID, inputs []SegmentRuleInput) []domain.SegmentRule {
	out := make([]domain.SegmentRule, len(inputs))
	for i, r := range inputs {
		out[i] = domain.SegmentRule{
			SegmentID: segmentID,
			Attribute: r.Attribute,
			Operator:  r.Operator,
			Value:     json.RawMessage(r.Value),
		}
	}
	return out
}
