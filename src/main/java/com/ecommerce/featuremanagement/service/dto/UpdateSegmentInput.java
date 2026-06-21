package com.ecommerce.featuremanagement.service.dto;

import com.ecommerce.featuremanagement.domain.AuditActor;
import com.ecommerce.featuremanagement.domain.SegmentOperator;

import java.util.List;

public class UpdateSegmentInput {
    public String name;
    public String description;
    public SegmentOperator operator;
    public List<SegmentRuleInput> rules;
    public AuditActor actor;
}
