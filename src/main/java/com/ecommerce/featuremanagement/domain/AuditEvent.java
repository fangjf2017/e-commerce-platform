package com.ecommerce.featuremanagement.domain;

import com.fasterxml.jackson.databind.JsonNode;
import java.time.Instant;
import java.util.HashMap;
import java.util.Map;
import java.util.UUID;

public class AuditEvent {
    public UUID id;
    public UUID applicationId;
    public UUID environmentId;
    public AuditAction action;
    public String resourceType; // feature_flag | rule | segment
    public UUID resourceId;
    public String resourceKey;
    public AuditActor actor;
    public JsonNode before;
    public JsonNode after;
    public Map<String, String> metadata = new HashMap<>();
    public Instant occurredAt;
}
