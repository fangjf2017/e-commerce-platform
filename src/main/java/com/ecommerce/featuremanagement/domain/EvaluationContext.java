package com.ecommerce.featuremanagement.domain;

import java.time.Instant;
import java.util.Map;
import java.util.UUID;

public class EvaluationContext {
    public String flagKey;
    public UUID applicationId;
    public UUID environmentId;
    public String entityId;
    public String entityType;
    public Map<String, Object> attributes;
    public String requestId;
    public Instant timestamp;
}
