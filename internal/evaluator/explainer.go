package evaluator

import (
	"fmt"

	"github.com/ecommerce/feature-management/internal/domain"
)

// BuildExplanation creates a human-readable explanation from the engine evaluation trace.
func BuildExplanation(
	flag *domain.FeatureFlag,
	envSlug string,
	evalCtx *domain.EvaluationContext,
	steps []domain.ExplanationStep,
	matchedRule *domain.Rule,
	bucketValue *int,
	defaultServed bool,
) domain.Explanation {
	exp := domain.Explanation{
		FlagKey:         flag.Key,
		FlagName:        flag.Name,
		FlagVersion:     flag.Version,
		EnvironmentSlug: envSlug,
		EvaluatedAt:     evalCtx.Timestamp,
		EntityID:        evalCtx.EntityID,
		EntityType:      evalCtx.EntityType,
		Steps:           steps,
		BucketValue:     bucketValue,
		DefaultServed:   defaultServed,
	}

	if matchedRule != nil {
		exp.MatchedRuleID = &matchedRule.ID
		exp.MatchedRuleName = matchedRule.Name
		exp.MatchedRuleType = matchedRule.Type
		exp.Reason = buildReason(matchedRule, bucketValue, evalCtx.EntityID)
	} else if defaultServed {
		exp.Reason = "No targeting rules matched; serving default value"
	} else if flag.Status == domain.FlagStatusInactive {
		exp.Reason = "Flag is disabled; serving default value"
	} else {
		exp.Reason = "Flag is archived"
	}

	return exp
}

func buildReason(rule *domain.Rule, bucket *int, entityID string) string {
	switch rule.Type {
	case domain.RuleTypeTargeting:
		return fmt.Sprintf("Matched targeting rule '%s' (all conditions satisfied)", rule.Name)
	case domain.RuleTypeSegment:
		return fmt.Sprintf("Entity is a member of segment targeted by rule '%s'", rule.Name)
	case domain.RuleTypeRollout:
		if bucket != nil && rule.RolloutPct != nil {
			return fmt.Sprintf("Entity bucket %d is within rollout threshold %d%% for rule '%s'", *bucket, *rule.RolloutPct*100, rule.Name)
		}
		return fmt.Sprintf("Matched rollout rule '%s'", rule.Name)
	case domain.RuleTypeSchedule:
		return fmt.Sprintf("Within active schedule window for rule '%s'", rule.Name)
	default:
		return fmt.Sprintf("Matched rule '%s'", rule.Name)
	}
}
