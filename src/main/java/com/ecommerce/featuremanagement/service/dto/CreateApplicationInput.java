package com.ecommerce.featuremanagement.service.dto;

import com.ecommerce.featuremanagement.domain.AuditActor;

public class CreateApplicationInput {
    public String name;
    public String slug;
    public String description;
    public AuditActor actor;
}
