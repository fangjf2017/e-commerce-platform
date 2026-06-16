package evaluator

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/ecommerce/feature-management/internal/cache"
	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/google/uuid"
)

// FlagRepository provides flag loading for cache misses.
type FlagRepository interface {
	GetByKey(ctx context.Context, appID, envID uuid.UUID, key string) (*domain.FeatureFlag, error)
	ListByEnvironment(ctx context.Context, appID, envID uuid.UUID) ([]*domain.FeatureFlag, error)
}

// SegmentRepository provides segment loading for cache misses.
type SegmentRepository interface {
	GetByID(ctx context.Context, appID, segID uuid.UUID) (*domain.Segment, error)
}

// EnvironmentRepository provides environment lookup for slug resolution.
type EnvironmentRepository interface {
	GetByID(ctx context.Context, appID, envID uuid.UUID) (*domain.Environment, error)
}

// Engine is the core feature flag evaluation engine. It resolves flags through
// the tiered cache and evaluates rules deterministically against the provided
// evaluation context.
type Engine struct {
	cache  *cache.TieredCache
	flagDB FlagRepository
	segDB  SegmentRepository
	envDB  EnvironmentRepository
}

func NewEngine(c *cache.TieredCache, flagDB FlagRepository, segDB SegmentRepository, envDB EnvironmentRepository) *Engine {
	return &Engine{cache: c, flagDB: flagDB, segDB: segDB, envDB: envDB}
}

// Evaluate resolves a single feature flag for the provided EvaluationContext.
// Rules are evaluated in priority order (ascending). The first matching rule
// wins. If no rule matches the flag's default value is returned.
func (e *Engine) Evaluate(ctx context.Context, evalCtx *domain.EvaluationContext) (*domain.EvaluationResult, error) {
	if evalCtx.Timestamp.IsZero() {
		evalCtx.Timestamp = time.Now()
	}

	flag, cacheLayer, err := e.cache.GetFlag(
		ctx,
		evalCtx.ApplicationID,
		evalCtx.EnvironmentID,
		evalCtx.FlagKey,
		func(ctx context.Context, appID, envID uuid.UUID, key string) (*domain.FeatureFlag, error) {
			return e.flagDB.GetByKey(ctx, appID, envID, key)
		},
	)
	if err != nil {
		return nil, err
	}

	cacheHit := cacheLayer != "db"

	// Resolve environment slug for the explanation (best-effort).
	envSlug := ""
	if env, err := e.envDB.GetByID(ctx, flag.ApplicationID, flag.EnvironmentID); err == nil {
		envSlug = env.Slug
	}

	// Sort a copy of the rules by priority (ascending) so priority=1 is evaluated first.
	rules := make([]domain.Rule, len(flag.Rules))
	copy(rules, flag.Rules)
	sort.Slice(rules, func(i, j int) bool { return rules[i].Priority < rules[j].Priority })

	steps := make([]domain.ExplanationStep, 0, len(rules))

	// If the flag is not active, return the default value immediately.
	if flag.Status != domain.FlagStatusActive {
		steps = append(steps, domain.ExplanationStep{
			Outcome: "flag_disabled",
		})
		return e.buildResult(flag, nil, flag.DefaultValue, "", steps, nil, true, cacheHit, cacheLayer, envSlug, evalCtx), nil
	}

	for i := range rules {
		rule := &rules[i]

		if !rule.Enabled {
			steps = append(steps, domain.ExplanationStep{
				RuleID:   &rule.ID,
				RuleName: rule.Name,
				RuleType: rule.Type,
				Priority: rule.Priority,
				Outcome:  "disabled",
			})
			continue
		}

		// Schedule window check — skip rule if we're outside the window.
		if rule.ScheduleStart != nil && evalCtx.Timestamp.Before(*rule.ScheduleStart) {
			steps = append(steps, domain.ExplanationStep{
				RuleID:   &rule.ID,
				RuleName: rule.Name,
				RuleType: rule.Type,
				Priority: rule.Priority,
				Outcome:  "schedule_miss",
			})
			continue
		}
		if rule.ScheduleEnd != nil && evalCtx.Timestamp.After(*rule.ScheduleEnd) {
			steps = append(steps, domain.ExplanationStep{
				RuleID:   &rule.ID,
				RuleName: rule.Name,
				RuleType: rule.Type,
				Priority: rule.Priority,
				Outcome:  "schedule_miss",
			})
			continue
		}

		step := domain.ExplanationStep{
			RuleID:   &rule.ID,
			RuleName: rule.Name,
			RuleType: rule.Type,
			Priority: rule.Priority,
		}

		matched := false

		switch rule.Type {
		case domain.RuleTypeTargeting:
			var condMatched bool
			condMatched, step.ConditionResults = MatchAllConditions(rule.Conditions, evalCtx.Attributes)
			matched = condMatched

		case domain.RuleTypeSegment:
			if rule.SegmentID != nil {
				seg, segErr := e.cache.GetSegment(
					ctx,
					flag.ApplicationID,
					*rule.SegmentID,
					func(ctx context.Context, appID, segID uuid.UUID) (*domain.Segment, error) {
						return e.segDB.GetByID(ctx, appID, segID)
					},
				)
				if segErr == nil {
					matched = MatchSegment(seg, evalCtx.Attributes)
					step.SegmentID = rule.SegmentID
					step.SegmentName = seg.Name
				}
			}

		case domain.RuleTypeRollout, domain.RuleTypeSchedule:
			// Conditions are optional guards for rollout/schedule rules.
			// If present, all must match before the rollout percentage is applied.
			if len(rule.Conditions) > 0 {
				condMatched, condResults := MatchAllConditions(rule.Conditions, evalCtx.Attributes)
				step.ConditionResults = condResults
				if !condMatched {
					step.Outcome = "skipped"
					steps = append(steps, step)
					continue
				}
			}
			matched = true
		}

		if !matched {
			step.Outcome = "skipped"
			steps = append(steps, step)
			continue
		}

		// Apply rollout percentage filter if configured.
		if rule.RolloutPct != nil {
			bucket := ComputeBucket(flag.Key, evalCtx.EntityID, rule.RolloutSalt)
			step.RolloutBucket = &bucket
			step.RolloutPct = rule.RolloutPct
			if !IsInRollout(bucket, *rule.RolloutPct) {
				step.Outcome = "skipped"
				steps = append(steps, step)
				continue
			}
		}

		// Rule matched — determine variant/value to serve.
		value, variantKey := e.selectVariantValue(flag, rule, evalCtx)
		step.Outcome = "matched"
		steps = append(steps, step)

		// Capture the bucket value for the explanation (compute again to keep it clean).
		var bucketVal *int
		if rule.RolloutPct != nil {
			b := ComputeBucket(flag.Key, evalCtx.EntityID, rule.RolloutSalt)
			bucketVal = &b
		}

		return e.buildResult(flag, rule, value, variantKey, steps, bucketVal, false, cacheHit, cacheLayer, envSlug, evalCtx), nil
	}

	// No rule matched — return the flag's default value.
	return e.buildResult(flag, nil, flag.DefaultValue, "", steps, nil, true, cacheHit, cacheLayer, envSlug, evalCtx), nil
}

// selectVariantValue determines which variant value to serve for the matched rule.
// If no variant allocations are defined on the rule, the first flag-level variant
// is returned (or the default value when no variants exist at all).
func (e *Engine) selectVariantValue(flag *domain.FeatureFlag, rule *domain.Rule, evalCtx *domain.EvaluationContext) (json.RawMessage, string) {
	if len(rule.Variants) == 0 {
		if len(flag.Variants) > 0 {
			return flag.Variants[0].Value, flag.Variants[0].Key
		}
		return flag.DefaultValue, ""
	}

	bucket := ComputeBucket(flag.Key, evalCtx.EntityID, rule.RolloutSalt)

	// Build a lookup map from variant ID to Variant for O(1) access.
	variantMap := make(map[uuid.UUID]*domain.Variant, len(flag.Variants))
	for i := range flag.Variants {
		variantMap[flag.Variants[i].ID] = &flag.Variants[i]
	}

	for _, alloc := range rule.Variants {
		if bucket >= alloc.RolloutFrom && bucket < alloc.RolloutTo {
			if v, ok := variantMap[alloc.VariantID]; ok {
				return v.Value, v.Key
			}
		}
	}

	return flag.DefaultValue, ""
}

// buildResult assembles the final EvaluationResult from the engine's outputs.
func (e *Engine) buildResult(
	flag *domain.FeatureFlag,
	matchedRule *domain.Rule,
	value json.RawMessage,
	variantKey string,
	steps []domain.ExplanationStep,
	bucketValue *int,
	defaultServed bool,
	cacheHit bool,
	cacheLayer string,
	envSlug string,
	evalCtx *domain.EvaluationContext,
) *domain.EvaluationResult {
	// For boolean flags, "enabled" reflects the actual boolean value served.
	// For all other types, enabled=true when a targeting rule matched (non-default).
	enabled := false
	if flag.Type == domain.FlagTypeBoolean {
		var b bool
		if err := json.Unmarshal(value, &b); err == nil {
			enabled = b
		}
	} else {
		enabled = !defaultServed
	}

	explanation := BuildExplanation(flag, envSlug, evalCtx, steps, matchedRule, bucketValue, defaultServed)

	return &domain.EvaluationResult{
		FlagKey:     flag.Key,
		FlagType:    flag.Type,
		Value:       value,
		VariantKey:  variantKey,
		Enabled:     enabled,
		Explanation: explanation,
		EvaluatedAt: evalCtx.Timestamp,
		CacheHit:    cacheHit,
		CacheLayer:  cacheLayer,
	}
}

// BatchEvaluate evaluates multiple flags in a single call for the same entity.
// When FlagKeys is empty, all active flags for the environment are evaluated.
// Errors on individual flags are skipped so partial results are still returned.
func (e *Engine) BatchEvaluate(ctx context.Context, req *domain.BatchEvaluationRequest) ([]*domain.EvaluationResult, error) {
	flagKeys := req.FlagKeys
	if len(flagKeys) == 0 {
		flags, err := e.flagDB.ListByEnvironment(ctx, req.ApplicationID, req.EnvironmentID)
		if err != nil {
			return nil, err
		}
		for _, f := range flags {
			flagKeys = append(flagKeys, f.Key)
		}
	}

	results := make([]*domain.EvaluationResult, 0, len(flagKeys))
	for _, key := range flagKeys {
		evalCtx := &domain.EvaluationContext{
			FlagKey:       key,
			ApplicationID: req.ApplicationID,
			EnvironmentID: req.EnvironmentID,
			EntityID:      req.EntityID,
			EntityType:    req.EntityType,
			Attributes:    req.Attributes,
			Timestamp:     time.Now(),
		}
		result, err := e.Evaluate(ctx, evalCtx)
		if err != nil {
			// Skip failed flags in batch; callers can detect missing keys.
			continue
		}
		results = append(results, result)
	}
	return results, nil
}
