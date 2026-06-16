package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AuditAction string

const (
	AuditActionFlagCreated       AuditAction = "flag.created"
	AuditActionFlagUpdated       AuditAction = "flag.updated"
	AuditActionFlagStatusChanged AuditAction = "flag.status_changed"
	AuditActionFlagDeleted       AuditAction = "flag.deleted"
	AuditActionRuleCreated       AuditAction = "rule.created"
	AuditActionRuleUpdated       AuditAction = "rule.updated"
	AuditActionRuleDeleted       AuditAction = "rule.deleted"
	AuditActionSegmentCreated    AuditAction = "segment.created"
	AuditActionSegmentUpdated    AuditAction = "segment.updated"
	AuditActionSegmentDeleted    AuditAction = "segment.deleted"
)

type AuditActor struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Type  string `json:"type"` // user | api_key | system
}

type AuditEvent struct {
	ID            uuid.UUID         `json:"id"`
	ApplicationID uuid.UUID         `json:"application_id"`
	EnvironmentID *uuid.UUID        `json:"environment_id,omitempty"`
	Action        AuditAction       `json:"action"`
	ResourceType  string            `json:"resource_type"`
	ResourceID    uuid.UUID         `json:"resource_id"`
	ResourceKey   string            `json:"resource_key"`
	Actor         AuditActor        `json:"actor"`
	Before        json.RawMessage   `json:"before,omitempty"`
	After         json.RawMessage   `json:"after,omitempty"`
	Metadata      map[string]string `json:"metadata"`
	OccurredAt    time.Time         `json:"occurred_at"`
}
