package com.ecommerce.featuremanagement.domain;

import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonValue;

public enum SegmentOperator {
    ALL("all"), ANY("any");

    private final String value;

    SegmentOperator(String value) { this.value = value; }

    @JsonValue
    public String value() { return value; }

    @JsonCreator
    public static SegmentOperator fromValue(String value) {
        for (SegmentOperator o : values()) {
            if (o.value.equals(value)) return o;
        }
        throw new IllegalArgumentException("unknown segment operator: " + value);
    }

    @Override
    public String toString() { return value; }
}
