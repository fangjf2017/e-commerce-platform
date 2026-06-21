package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type SegmentOperator string

const (
	SegmentOpAll SegmentOperator = "all"
	SegmentOpAny SegmentOperator = "any"
)

type Segment struct {
	ID            uuid.UUID       `json:"id"`
	ApplicationID uuid.UUID       `json:"application_id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Operator      SegmentOperator `json:"operator"`
	Rules         []SegmentRule   `json:"rules"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	Version       int64           `json:"version"`
}

type SegmentRule struct {
	ID        uuid.UUID         `json:"id"`
	SegmentID uuid.UUID         `json:"segment_id"`
	Attribute string            `json:"attribute"`
	Operator  ConditionOperator `json:"operator"`
	Value     json.RawMessage   `json:"value"`
}
