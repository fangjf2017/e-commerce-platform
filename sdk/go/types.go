package featureflags

import (
	"encoding/json"
	"time"
)

type EvaluationResult struct {
	FlagKey     string          `json:"flag_key"`
	FlagType    string          `json:"flag_type"`
	Value       json.RawMessage `json:"value"`
	VariantKey  string          `json:"variant_key,omitempty"`
	Enabled     bool            `json:"enabled"`
	Explanation Explanation     `json:"explanation"`
	EvaluatedAt time.Time       `json:"evaluated_at"`
	CacheHit    bool            `json:"cache_hit"`
	CacheLayer  string          `json:"cache_layer,omitempty"`
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
	MatchedRuleID   *string           `json:"matched_rule_id,omitempty"`
	MatchedRuleName string            `json:"matched_rule_name,omitempty"`
	MatchedRuleType string            `json:"matched_rule_type,omitempty"`
	BucketValue     *int              `json:"bucket_value,omitempty"`
	DefaultServed   bool              `json:"default_served"`
	Reason          string            `json:"reason"`
}

type ExplanationStep struct {
	RuleID           *string           `json:"rule_id,omitempty"`
	RuleName         string            `json:"rule_name"`
	RuleType         string            `json:"rule_type"`
	Priority         int               `json:"priority"`
	Outcome          string            `json:"outcome"`
	ConditionResults []ConditionResult `json:"condition_results,omitempty"`
	SegmentID        *string           `json:"segment_id,omitempty"`
	SegmentName      string            `json:"segment_name,omitempty"`
	RolloutBucket    *int              `json:"rollout_bucket,omitempty"`
	RolloutPct       *int              `json:"rollout_pct,omitempty"`
}

type ConditionResult struct {
	Attribute   string      `json:"attribute"`
	Operator    string      `json:"operator"`
	Value       interface{} `json:"value"`
	ActualValue interface{} `json:"actual_value"`
	Matched     bool        `json:"matched"`
}

type EvalContext struct {
	ApplicationID string                 `json:"application_id"`
	EnvironmentID string                 `json:"environment_id"`
	EntityID      string                 `json:"entity_id"`
	EntityType    string                 `json:"entity_type"`
	Attributes    map[string]interface{} `json:"attributes"`
}
