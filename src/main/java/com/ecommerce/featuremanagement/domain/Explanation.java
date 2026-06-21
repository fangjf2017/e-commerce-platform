package com.ecommerce.featuremanagement.domain;

import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.UUID;

public class Explanation {
    public String flagKey;
    public String flagName;
    public long flagVersion;
    public String environmentSlug;
    public Instant evaluatedAt;
    public String entityId;
    public String entityType;
    public List<ExplanationStep> steps = new ArrayList<>();
    public UUID matchedRuleId;
    public String matchedRuleName;
    public RuleType matchedRuleType;
    public Integer bucketValue;
    public boolean defaultServed;
    public String reason;
}
