package com.ecommerce.featuremanagement.domain;

import com.fasterxml.jackson.databind.JsonNode;
import java.util.UUID;

public class SegmentRule {
    public UUID id;
    public UUID segmentId;
    public String attribute;
    public ConditionOperator operator;
    public JsonNode value;
}
