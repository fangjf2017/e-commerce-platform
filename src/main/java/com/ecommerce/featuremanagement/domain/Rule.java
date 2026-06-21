package com.ecommerce.featuremanagement.domain;

import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.UUID;

public class Rule {
    public UUID id;
    public UUID flagId;
    public RuleType type;
    public int priority;
    public String name;
    public String description;
    public List<Condition> conditions = new ArrayList<>();
    public UUID segmentId;
    public Integer rolloutPct;
    public UUID rolloutSalt;
    public List<VariantAllocation> variants = new ArrayList<>();
    public Instant scheduleStart;
    public Instant scheduleEnd;
    public boolean enabled = true;
    public Instant createdAt;
    public Instant updatedAt;
}
