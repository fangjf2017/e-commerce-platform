package evaluator

import (
	"encoding/json"

	"github.com/ecommerce/feature-management/internal/domain"
)

// MatchSegment returns true if attributes satisfy the segment's membership rules.
// Applies SegmentOpAll (AND) or SegmentOpAny (OR) across all segment rules.
// An empty rule set never matches.
func MatchSegment(seg *domain.Segment, attributes map[string]interface{}) bool {
	if len(seg.Rules) == 0 {
		return false
	}
	for _, rule := range seg.Rules {
		cond := domain.Condition{
			ID:        rule.ID,
			Attribute: rule.Attribute,
			Operator:  rule.Operator,
			Value:     rule.Value,
		}
		matched, _ := MatchCondition(cond, attributes)
		if seg.Operator == domain.SegmentOpAny && matched {
			return true
		}
		if seg.Operator == domain.SegmentOpAll && !matched {
			return false
		}
	}
	// For ALL: we reached the end without a failure → all matched.
	// For ANY: we reached the end without any match → none matched.
	return seg.Operator == domain.SegmentOpAll
}

// MatchAllConditions returns true if ALL conditions in a rule match, along with
// per-condition explanation details for audit / debug purposes.
func MatchAllConditions(conditions []domain.Condition, attributes map[string]interface{}) (bool, []domain.ExplanationMatchedCondition) {
	results := make([]domain.ExplanationMatchedCondition, 0, len(conditions))
	allMatched := true
	for _, cond := range conditions {
		matched, actual := MatchCondition(cond, attributes)
		var condValue interface{}
		_ = json.Unmarshal(cond.Value, &condValue)
		results = append(results, domain.ExplanationMatchedCondition{
			ConditionID: cond.ID,
			Attribute:   cond.Attribute,
			Operator:    cond.Operator,
			Value:       condValue,
			ActualValue: actual,
			Matched:     matched,
		})
		if !matched {
			allMatched = false
		}
	}
	return allMatched, results
}
