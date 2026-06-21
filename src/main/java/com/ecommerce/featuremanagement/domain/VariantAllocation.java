package com.ecommerce.featuremanagement.domain;

import java.util.UUID;

public class VariantAllocation {
    public UUID id;
    public UUID ruleId;
    public UUID variantId;
    public int rolloutFrom;
    public int rolloutTo;
}
