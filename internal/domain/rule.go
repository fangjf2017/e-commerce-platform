package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type RuleType string

const (
	RuleTypeTargeting RuleType = "targeting"
	RuleTypeSegment   RuleType = "segment"
	RuleTypeRollout   RuleType = "rollout"
	RuleTypeSchedule  RuleType = "schedule"
)

type ConditionOperator string

const (
	OpEquals         ConditionOperator = "eq"
	OpNotEquals      ConditionOperator = "neq"
	OpContains       ConditionOperator = "contains"
	OpNotContains    ConditionOperator = "not_contains"
	OpStartsWith     ConditionOperator = "starts_with"
	OpEndsWith       ConditionOperator = "ends_with"
	OpGreaterThan    ConditionOperator = "gt"
	OpGreaterOrEqual ConditionOperator = "gte"
	OpLessThan       ConditionOperator = "lt"
	OpLessOrEqual    ConditionOperator = "lte"
	OpIn             ConditionOperator = "in"
	OpNotIn          ConditionOperator = "not_in"
	OpRegex          ConditionOperator = "regex"
	OpSemverGte      ConditionOperator = "semver_gte"
	OpExists         ConditionOperator = "exists"
	OpNotExists      ConditionOperator = "not_exists"
)

type Condition struct {
	ID        uuid.UUID         `json:"id"`
	RuleID    uuid.UUID         `json:"rule_id"`
	Attribute string            `json:"attribute"`
	Operator  ConditionOperator `json:"operator"`
	Value     json.RawMessage   `json:"value"`
	Negate    bool              `json:"negate"`
}

type Rule struct {
	ID            uuid.UUID           `json:"id"`
	FlagID        uuid.UUID           `json:"flag_id"`
	Type          RuleType            `json:"type"`
	Priority      int                 `json:"priority"`
	Name          string              `json:"name"`
	Description   string              `json:"description"`
	Conditions    []Condition         `json:"conditions"`
	SegmentID     *uuid.UUID          `json:"segment_id,omitempty"`
	RolloutPct    *int                `json:"rollout_pct,omitempty"`
	RolloutSalt   uuid.UUID           `json:"rollout_salt"`
	Variants      []VariantAllocation `json:"variants"`
	ScheduleStart *time.Time          `json:"schedule_start,omitempty"`
	ScheduleEnd   *time.Time          `json:"schedule_end,omitempty"`
	Enabled       bool                `json:"enabled"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
}
