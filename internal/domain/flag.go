package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type FlagType string

const (
	FlagTypeBoolean FlagType = "boolean"
	FlagTypeString  FlagType = "string"
	FlagTypeNumber  FlagType = "number"
	FlagTypeJSON    FlagType = "json"
)

type FlagStatus string

const (
	FlagStatusActive   FlagStatus = "active"
	FlagStatusInactive FlagStatus = "inactive"
	FlagStatusArchived FlagStatus = "archived"
)

type FeatureFlag struct {
	ID            uuid.UUID       `json:"id"`
	ApplicationID uuid.UUID       `json:"application_id"`
	EnvironmentID uuid.UUID       `json:"environment_id"`
	Key           string          `json:"key"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Type          FlagType        `json:"type"`
	Status        FlagStatus      `json:"status"`
	DefaultValue  json.RawMessage `json:"default_value"`
	Tags          []string        `json:"tags"`
	Rules         []Rule          `json:"rules"`
	Variants      []Variant       `json:"variants"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	Version       int64           `json:"version"`
}

type Variant struct {
	ID          uuid.UUID       `json:"id"`
	FlagID      uuid.UUID       `json:"flag_id"`
	Key         string          `json:"key"`
	Value       json.RawMessage `json:"value"`
	Description string          `json:"description"`
}

type VariantAllocation struct {
	ID          uuid.UUID `json:"id"`
	RuleID      uuid.UUID `json:"rule_id"`
	VariantID   uuid.UUID `json:"variant_id"`
	RolloutFrom int       `json:"rollout_from"`
	RolloutTo   int       `json:"rollout_to"`
}
