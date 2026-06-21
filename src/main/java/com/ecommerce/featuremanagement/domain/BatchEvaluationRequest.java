package com.ecommerce.featuremanagement.domain;

import java.util.List;
import java.util.Map;
import java.util.UUID;

public class BatchEvaluationRequest {
    public UUID applicationId;
    public UUID environmentId;
    public String entityId;
    public String entityType;
    public Map<String, Object> attributes;
    public List<String> flagKeys;
}
