package com.ecommerce.featuremanagement.domain;

import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonValue;

public enum ConditionOperator {
    EQ("eq"), NEQ("neq"), CONTAINS("contains"), NOT_CONTAINS("not_contains"),
    STARTS_WITH("starts_with"), ENDS_WITH("ends_with"),
    GT("gt"), GTE("gte"), LT("lt"), LTE("lte"),
    IN("in"), NOT_IN("not_in"), REGEX("regex"), SEMVER_GTE("semver_gte"),
    EXISTS("exists"), NOT_EXISTS("not_exists");

    private final String value;

    ConditionOperator(String value) { this.value = value; }

    @JsonValue
    public String value() { return value; }

    @JsonCreator
    public static ConditionOperator fromValue(String value) {
        for (ConditionOperator o : values()) {
            if (o.value.equals(value)) return o;
        }
        throw new IllegalArgumentException("unknown condition operator: " + value);
    }

    @Override
    public String toString() { return value; }
}
