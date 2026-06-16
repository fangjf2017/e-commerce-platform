package evaluator

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ecommerce/feature-management/internal/domain"
)

// MatchCondition evaluates a single condition against the provided attributes.
// It returns whether the condition matched and the actual attribute value
// (for use in evaluation explanations). The condition's Negate flag is applied
// after the base operator result is computed.
func MatchCondition(condition domain.Condition, attributes map[string]interface{}) (matched bool, actualValue interface{}) {
	// exists / not_exists are special: they don't need the attribute value.
	op := condition.Operator
	if op == domain.OpExists || op == domain.OpNotExists {
		val, present := attributes[condition.Attribute]
		actualValue = val
		isPresent := present && val != nil
		result := false
		if op == domain.OpExists {
			result = isPresent
		} else {
			result = !isPresent
		}
		if condition.Negate {
			result = !result
		}
		return result, actualValue
	}

	// Retrieve the actual attribute value.
	attrVal, present := attributes[condition.Attribute]
	actualValue = attrVal

	// Unmarshal the condition's expected value from JSON.
	var condValue interface{}
	if len(condition.Value) > 0 {
		if err := json.Unmarshal(condition.Value, &condValue); err != nil {
			// If we can't parse the condition value, the condition cannot match.
			return applyNegate(false, condition.Negate), actualValue
		}
	}

	if !present || attrVal == nil {
		// Attribute missing — no operator (except exists/not_exists handled above) can match.
		return applyNegate(false, condition.Negate), actualValue
	}

	var result bool
	switch op {
	case domain.OpEquals:
		result = equalValues(attrVal, condValue)

	case domain.OpNotEquals:
		result = !equalValues(attrVal, condValue)

	case domain.OpContains:
		result = stringOp(attrVal, condValue, func(a, b string) bool {
			return strings.Contains(a, b)
		})

	case domain.OpNotContains:
		result = stringOp(attrVal, condValue, func(a, b string) bool {
			return !strings.Contains(a, b)
		})

	case domain.OpStartsWith:
		result = stringOp(attrVal, condValue, func(a, b string) bool {
			return strings.HasPrefix(a, b)
		})

	case domain.OpEndsWith:
		result = stringOp(attrVal, condValue, func(a, b string) bool {
			return strings.HasSuffix(a, b)
		})

	case domain.OpGreaterThan:
		result = numericOp(attrVal, condValue, func(a, b float64) bool { return a > b })

	case domain.OpGreaterOrEqual:
		result = numericOp(attrVal, condValue, func(a, b float64) bool { return a >= b })

	case domain.OpLessThan:
		result = numericOp(attrVal, condValue, func(a, b float64) bool { return a < b })

	case domain.OpLessOrEqual:
		result = numericOp(attrVal, condValue, func(a, b float64) bool { return a <= b })

	case domain.OpIn:
		result = inOperator(attrVal, condValue)

	case domain.OpNotIn:
		result = !inOperator(attrVal, condValue)

	case domain.OpRegex:
		result = regexMatch(attrVal, condValue)

	case domain.OpSemverGte:
		result = semverGte(attrVal, condValue)

	default:
		// Unknown operator — treat as no match.
		result = false
	}

	return applyNegate(result, condition.Negate), actualValue
}

// applyNegate flips the result when negate is true.
func applyNegate(result, negate bool) bool {
	if negate {
		return !result
	}
	return result
}

// toFloat64 converts a value to float64. Handles json.Number, float32/64,
// int/int64/int32/int16/int8, uint variants, and string representations.
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int8:
		return float64(val), true
	case int16:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint8:
		return float64(val), true
	case uint16:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint64:
		return float64(val), true
	case json.Number:
		f, err := val.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(val, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

// toString converts a value to its string representation.
func toString(v interface{}) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true
	case fmt.Stringer:
		return val.String(), true
	default:
		return fmt.Sprintf("%v", v), true
	}
}

// equalValues performs type-coercing equality comparison.
// Numbers are compared as float64; everything else is compared as strings.
func equalValues(a, b interface{}) bool {
	// Try numeric comparison first.
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if aOk && bOk {
		return af == bf
	}

	// Boolean comparison.
	if ab, ok := a.(bool); ok {
		if bb, ok := b.(bool); ok {
			return ab == bb
		}
	}

	// Fall back to string comparison.
	as, _ := toString(a)
	bs, _ := toString(b)
	return as == bs
}

// stringOp applies a string operation between the attribute value and condition value.
// Both sides are coerced to strings.
func stringOp(attrVal, condVal interface{}, fn func(a, b string) bool) bool {
	a, ok1 := toString(attrVal)
	b, ok2 := toString(condVal)
	if !ok1 || !ok2 {
		return false
	}
	return fn(a, b)
}

// numericOp applies a numeric comparison between the attribute value and condition value.
func numericOp(attrVal, condVal interface{}, fn func(a, b float64) bool) bool {
	a, ok1 := toFloat64(attrVal)
	b, ok2 := toFloat64(condVal)
	if !ok1 || !ok2 {
		return false
	}
	return fn(a, b)
}

// inOperator returns true if attrVal is found within the condVal array.
// condVal must unmarshal as []interface{}.
func inOperator(attrVal, condVal interface{}) bool {
	items, ok := toSlice(condVal)
	if !ok {
		return false
	}
	for _, item := range items {
		if equalValues(attrVal, item) {
			return true
		}
	}
	return false
}

// toSlice converts a value to []interface{} if possible.
func toSlice(v interface{}) ([]interface{}, bool) {
	switch val := v.(type) {
	case []interface{}:
		return val, true
	default:
		return nil, false
	}
}

// regexMatch compiles the condition value as a regex and tests the attribute string.
func regexMatch(attrVal, condVal interface{}) bool {
	pattern, ok := toString(condVal)
	if !ok {
		return false
	}
	subject, ok := toString(attrVal)
	if !ok {
		return false
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(subject)
}

// parseSemver parses a version string "X.Y.Z" into [3]int.
// Missing components default to 0. Extra components are ignored.
func parseSemver(s string) ([3]int, bool) {
	parts := strings.SplitN(strings.TrimSpace(s), ".", 4)
	var v [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		n, err := strconv.Atoi(strings.TrimSpace(parts[i]))
		if err != nil {
			return v, false
		}
		v[i] = n
	}
	return v, true
}

// semverGte returns true if the attribute version is >= the condition version.
// Both must be parseable as "X.Y.Z" version strings.
func semverGte(attrVal, condVal interface{}) bool {
	attrStr, ok1 := toString(attrVal)
	condStr, ok2 := toString(condVal)
	if !ok1 || !ok2 {
		return false
	}
	av, ok1 := parseSemver(attrStr)
	cv, ok2 := parseSemver(condStr)
	if !ok1 || !ok2 {
		return false
	}
	// Compare major, then minor, then patch.
	for i := 0; i < 3; i++ {
		if av[i] > cv[i] {
			return true
		}
		if av[i] < cv[i] {
			return false
		}
	}
	return true // equal
}
