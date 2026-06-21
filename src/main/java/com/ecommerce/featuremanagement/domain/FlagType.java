package com.ecommerce.featuremanagement.domain;

import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonValue;

public enum FlagType {
    BOOLEAN("boolean"), STRING("string"), NUMBER("number"), JSON("json");

    private final String value;

    FlagType(String value) { this.value = value; }

    @JsonValue
    public String value() { return value; }

    @JsonCreator
    public static FlagType fromValue(String value) {
        for (FlagType t : values()) {
            if (t.value.equals(value)) return t;
        }
        throw new IllegalArgumentException("unknown flag type: " + value);
    }

    @Override
    public String toString() { return value; }
}
