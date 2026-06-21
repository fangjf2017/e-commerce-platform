package com.ecommerce.featuremanagement.service.dto;

import com.ecommerce.featuremanagement.domain.ConditionOperator;
import com.fasterxml.jackson.databind.JsonNode;

public class ConditionInput {
    public String attribute;
    public ConditionOperator operator;
    public JsonNode value;
    public boolean negate;
}
