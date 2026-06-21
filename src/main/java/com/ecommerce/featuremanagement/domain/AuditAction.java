package com.ecommerce.featuremanagement.domain;

import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonValue;

public enum AuditAction {
    FLAG_CREATED("flag.created"), FLAG_UPDATED("flag.updated"),
    FLAG_STATUS_CHANGED("flag.status_changed"), FLAG_DELETED("flag.deleted"),
    RULE_CREATED("rule.created"), RULE_UPDATED("rule.updated"), RULE_DELETED("rule.deleted"),
    SEGMENT_CREATED("segment.created"), SEGMENT_UPDATED("segment.updated"), SEGMENT_DELETED("segment.deleted");

    private final String value;

    AuditAction(String value) { this.value = value; }

    @JsonValue
    public String value() { return value; }

    @JsonCreator
    public static AuditAction fromValue(String value) {
        for (AuditAction a : values()) {
            if (a.value.equals(value)) return a;
        }
        throw new IllegalArgumentException("unknown audit action: " + value);
    }

    @Override
    public String toString() { return value; }
}
