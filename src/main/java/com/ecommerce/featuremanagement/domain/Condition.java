package com.ecommerce.featuremanagement.domain;

import com.fasterxml.jackson.databind.JsonNode;
import java.util.UUID;

public class Condition {
    public UUID id;
    public UUID ruleId;
    public String attribute;
    public ConditionOperator operator;
    public JsonNode value;
    public boolean negate;
}
