package com.ecommerce.featuremanagement.domain;

import com.fasterxml.jackson.databind.JsonNode;
import java.time.Instant;

public class EvaluationResult {
    public String flagKey;
    public FlagType flagType;
    public JsonNode value;
    public String variantKey;
    public boolean enabled;
    public Explanation explanation;
    public Instant evaluatedAt;
    public boolean cacheHit;
    public String cacheLayer; // l1 | l2 | db
}
