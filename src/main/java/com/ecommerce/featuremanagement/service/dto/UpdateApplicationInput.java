package com.ecommerce.featuremanagement.service.dto;

import com.ecommerce.featuremanagement.domain.AuditActor;

public class UpdateApplicationInput {
    public String name;
    public String description;
    public AuditActor actor;
}
