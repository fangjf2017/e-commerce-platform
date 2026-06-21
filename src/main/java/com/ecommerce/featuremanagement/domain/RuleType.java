package com.ecommerce.featuremanagement.domain;

import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonValue;

public enum RuleType {
    TARGETING("targeting"), SEGMENT("segment"), ROLLOUT("rollout"), SCHEDULE("schedule");

    private final String value;

    RuleType(String value) { this.value = value; }

    @JsonValue
    public String value() { return value; }

    @JsonCreator
    public static RuleType fromValue(String value) {
        for (RuleType t : values()) {
            if (t.value.equals(value)) return t;
        }
        throw new IllegalArgumentException("unknown rule type: " + value);
    }

    @Override
    public String toString() { return value; }
}
