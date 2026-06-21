package com.ecommerce.featuremanagement.domain;

import java.util.ArrayList;
import java.util.List;
import java.util.UUID;

public class ExplanationStep {
    public UUID ruleId;
    public String ruleName;
    public RuleType ruleType;
    public int priority;
    public List<ExplanationMatchedCondition> conditionResults = new ArrayList<>();
    public UUID segmentId;
    public String segmentName;
    public Integer rolloutBucket;
    public Integer rolloutPct;
    public String outcome; // matched | skipped | schedule_miss | disabled | flag_disabled
}
