package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type EvaluationContext struct {
	FlagKey       string                 `json:"flag_key"`
	ApplicationID uuid.UUID              `json:"application_id"`
	EnvironmentID uuid.UUID              `json:"environment_id"`
	EntityID      string                 `json:"entity_id"`
	EntityType    string                 `json:"entity_type"`
	Attributes    map[string]interface{} `json:"attributes"`
	RequestID     string                 `json:"request_id"`
	Timestamp     time.Time              `json:"timestamp"`
}

type BatchEvaluationRequest struct {
	ApplicationID uuid.UUID              `json:"application_id"`
	EnvironmentID uuid.UUID              `json:"environment_id"`
	EntityID      string                 `json:"entity_id"`
	EntityType    string                 `json:"entity_type"`
	Attributes    map[string]interface{} `json:"attributes"`
	FlagKeys      []string               `json:"flag_keys"`
}

type ExplanationMatchedCondition struct {
	ConditionID uuid.UUID         `json:"condition_id"`
	Attribute   string            `json:"attribute"`
	Operator    ConditionOperator `json:"operator"`
	Value       interface{}       `json:"value"`
	ActualValue interface{}       `json:"actual_value"`
	Matched     bool              `json:"matched"`
}

type ExplanationStep struct {
	RuleID           *uuid.UUID                    `json:"rule_id,omitempty"`
	RuleName         string                        `json:"rule_name"`
	RuleType         RuleType                      `json:"rule_type"`
	Priority         int                           `json:"priority"`
	ConditionResults []ExplanationMatchedCondition `json:"condition_results,omitempty"`
	SegmentID        *uuid.UUID                    `json:"segment_id,omitempty"`
	SegmentName      string                        `json:"segment_name,omitempty"`
	RolloutBucket    *int                          `json:"rollout_bucket,omitempty"`
	RolloutPct       *int                          `json:"rollout_pct,omitempty"`
	Outcome          string                        `json:"outcome"` // matched | skipped | schedule_miss | disabled
}

type Explanation struct {
	FlagKey         string            `json:"flag_key"`
	FlagName        string            `json:"flag_name"`
	FlagVersion     int64             `json:"flag_version"`
	EnvironmentSlug string            `json:"environment_slug"`
	EvaluatedAt     time.Time         `json:"evaluated_at"`
	EntityID        string            `json:"entity_id"`
	EntityType      string            `json:"entity_type"`
	Steps           []ExplanationStep `json:"steps"`
	MatchedRuleID   *uuid.UUID        `json:"matched_rule_id,omitempty"`
	MatchedRuleName string            `json:"matched_rule_name,omitempty"`
	MatchedRuleType RuleType          `json:"matched_rule_type,omitempty"`
	BucketValue     *int              `json:"bucket_value,omitempty"`
	DefaultServed   bool              `json:"default_served"`
	Reason          string            `json:"reason"`
}

type EvaluationResult struct {
	FlagKey     string          `json:"flag_key"`
	FlagType    FlagType        `json:"flag_type"`
	Value       json.RawMessage `json:"value"`
	VariantKey  string          `json:"variant_key,omitempty"`
	Enabled     bool            `json:"enabled"`
	Explanation Explanation     `json:"explanation"`
	EvaluatedAt time.Time       `json:"evaluated_at"`
	CacheHit    bool            `json:"cache_hit"`
	CacheLayer  string          `json:"cache_layer,omitempty"`
}
