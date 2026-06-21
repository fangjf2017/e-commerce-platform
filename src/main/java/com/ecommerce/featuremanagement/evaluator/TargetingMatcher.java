package com.ecommerce.featuremanagement.evaluator;

import com.ecommerce.featuremanagement.domain.Condition;
import com.ecommerce.featuremanagement.domain.Segment;
import com.ecommerce.featuremanagement.domain.SegmentOperator;
import com.ecommerce.featuremanagement.domain.SegmentRule;

import java.util.List;
import java.util.Map;

public final class TargetingMatcher {

    private TargetingMatcher() {}

    public static boolean matchAllConditions(List<Condition> conditions, Map<String, Object> attributes) {
        if (conditions == null || conditions.isEmpty()) {
            return true;
        }
        for (Condition c : conditions) {
            if (!ConditionMatcher.matches(c, attributes)) {
                return false;
            }
        }
        return true;
    }

    public static boolean matchSegment(Segment segment, Map<String, Object> attributes) {
        if (segment == null || segment.rules == null || segment.rules.isEmpty()) {
            return false;
        }
        if (segment.operator == SegmentOperator.ANY) {
            for (SegmentRule r : segment.rules) {
                if (ConditionMatcher.matches(r, attributes)) {
                    return true;
                }
            }
            return false;
        }
        // SegmentOperator.ALL
        for (SegmentRule r : segment.rules) {
            if (!ConditionMatcher.matches(r, attributes)) {
                return false;
            }
        }
        return true;
    }
}
