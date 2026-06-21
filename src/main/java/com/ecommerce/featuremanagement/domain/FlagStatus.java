package com.ecommerce.featuremanagement.domain;

import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonValue;

public enum FlagStatus {
    ACTIVE("active"), INACTIVE("inactive"), ARCHIVED("archived");

    private final String value;

    FlagStatus(String value) { this.value = value; }

    @JsonValue
    public String value() { return value; }

    @JsonCreator
    public static FlagStatus fromValue(String value) {
        for (FlagStatus s : values()) {
            if (s.value.equals(value)) return s;
        }
        throw new IllegalArgumentException("unknown flag status: " + value);
    }

    @Override
    public String toString() { return value; }
}
