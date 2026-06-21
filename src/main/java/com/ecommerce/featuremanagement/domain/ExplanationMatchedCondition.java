package com.ecommerce.featuremanagement.domain;

import java.util.UUID;

public class ExplanationMatchedCondition {
    public UUID conditionId;
    public String attribute;
    public ConditionOperator operator;
    public Object value;
    public Object actualValue;
    public boolean matched;
}
