package com.ecommerce.featuremanagement.service.dto;

import com.ecommerce.featuremanagement.domain.AuditActor;
import com.ecommerce.featuremanagement.domain.RuleType;

import java.time.Instant;
import java.util.List;
import java.util.UUID;

public class UpdateRuleInput {
    public RuleType type;
    public int priority;
    public String name;
    public String description;
    public List<ConditionInput> conditions;
    public UUID segmentId;
    public Integer rolloutPct;
    public List<VariantAllocationInput> variants;
    public Instant scheduleStart;
    public Instant scheduleEnd;
    public boolean enabled = true;
    public AuditActor actor;
}
