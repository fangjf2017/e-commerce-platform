package com.ecommerce.featuremanagement.service.dto;

import java.util.UUID;

public class VariantAllocationInput {
    public UUID variantId;
    public int rolloutFrom;
    public int rolloutTo;
}
