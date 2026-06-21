package com.ecommerce.featuremanagement.evaluator;

import com.ecommerce.featuremanagement.domain.Condition;
import com.ecommerce.featuremanagement.domain.ConditionOperator;
import com.ecommerce.featuremanagement.domain.SegmentRule;
import com.fasterxml.jackson.databind.JsonNode;

import java.util.Iterator;
import java.util.Map;
import java.util.regex.Pattern;

/** Implements the 16 ConditionOperator semantics, matching the Go evaluator/condition_matcher.go. */
public final class ConditionMatcher {

    private ConditionMatcher() {}

    public static boolean matches(String attribute, ConditionOperator operator, JsonNode value, boolean negate,
                                   Map<String, Object> attributes) {
        boolean result = evaluate(attribute, operator, value, attributes);
        return negate ? !result : result;
    }

    public static boolean matches(Condition c, Map<String, Object> attributes) {
        return matches(c.attribute, c.operator, c.value, c.negate, attributes);
    }

    public static boolean matches(SegmentRule r, Map<String, Object> attributes) {
        return matches(r.attribute, r.operator, r.value, false, attributes);
    }

    private static boolean evaluate(String attribute, ConditionOperator operator, JsonNode value,
                                      Map<String, Object> attributes) {
        boolean present = attributes != null && attributes.containsKey(attribute) && attributes.get(attribute) != null;

        if (operator == ConditionOperator.EXISTS) {
            return present;
        }
        if (operator == ConditionOperator.NOT_EXISTS) {
            return !present;
        }
        if (!present) {
            return false;
        }

        Object actual = attributes.get(attribute);

        switch (operator) {
            case EQ:
                return equalValues(actual, value);
            case NEQ:
                return !equalValues(actual, value);
            case CONTAINS:
                return toStr(actual).contains(toStr(value));
            case NOT_CONTAINS:
                return !toStr(actual).contains(toStr(value));
            case STARTS_WITH:
                return toStr(actual).startsWith(toStr(value));
            case ENDS_WITH:
                return toStr(actual).endsWith(toStr(value));
            case GT:
            case GTE:
            case LT:
            case LTE: {
                Double a = toDouble(actual);
                Double b = toDouble(value);
                if (a == null || b == null) {
                    return false;
                }
                switch (operator) {
                    case GT: return a > b;
                    case GTE: return a >= b;
                    case LT: return a < b;
                    default: return a <= b;
                }
            }
            case IN:
            case NOT_IN: {
                boolean found = false;
                if (value != null && value.isArray()) {
                    Iterator<JsonNode> it = value.elements();
                    while (it.hasNext()) {
                        if (equalValues(actual, it.next())) {
                            found = true;
                            break;
                        }
                    }
                }
                return operator == ConditionOperator.IN ? found : !found;
            }
            case REGEX: {
                try {
                    Pattern p = Pattern.compile(toStr(value));
                    return p.matcher(toStr(actual)).find();
                } catch (Exception e) {
                    return false;
                }
            }
            case SEMVER_GTE:
                return semverGte(toStr(actual), toStr(value));
            default:
                return false;
        }
    }

    private static boolean equalValues(Object actual, JsonNode expected) {
        Double an = toDouble(actual);
        Double bn = toDouble(expected);
        if (an != null && bn != null) {
            return an.doubleValue() == bn.doubleValue();
        }
        if (actual instanceof Boolean && expected != null && expected.isBoolean()) {
            return actual.equals(expected.asBoolean());
        }
        return toStr(actual).equals(toStr(expected));
    }

    private static Double toDouble(Object o) {
        if (o == null) {
            return null;
        }
        if (o instanceof Number) {
            return ((Number) o).doubleValue();
        }
        if (o instanceof JsonNode) {
            JsonNode n = (JsonNode) o;
            if (n.isNumber()) {
                return n.asDouble();
            }
            if (n.isTextual()) {
                try {
                    return Double.parseDouble(n.asText());
                } catch (NumberFormatException e) {
                    return null;
                }
            }
            return null;
        }
        if (o instanceof String) {
            try {
                return Double.parseDouble((String) o);
            } catch (NumberFormatException e) {
                return null;
            }
        }
        return null;
    }

    private static String toStr(Object o) {
        if (o == null) {
            return "";
        }
        if (o instanceof JsonNode) {
            JsonNode n = (JsonNode) o;
            return n.isTextual() ? n.asText() : n.toString();
        }
        return String.valueOf(o);
    }

    private static boolean semverGte(String actual, String expected) {
        int[] a = parseSemver(actual);
        int[] b = parseSemver(expected);
        for (int i = 0; i < 3; i++) {
            if (a[i] != b[i]) {
                return a[i] > b[i];
            }
        }
        return true;
    }

    private static int[] parseSemver(String s) {
        int[] parts = new int[]{0, 0, 0};
        if (s == null) {
            return parts;
        }
        String[] split = s.split("\\.");
        for (int i = 0; i < 3 && i < split.length; i++) {
            try {
                parts[i] = Integer.parseInt(split[i].replaceAll("[^0-9].*$", ""));
            } catch (NumberFormatException e) {
                parts[i] = 0;
            }
        }
        return parts;
    }
}
