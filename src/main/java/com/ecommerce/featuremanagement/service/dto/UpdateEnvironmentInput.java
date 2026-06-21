package com.ecommerce.featuremanagement.service.dto;

import com.ecommerce.featuremanagement.domain.AuditActor;

public class UpdateEnvironmentInput {
    public String name;
    public boolean requiresApproval;
    public AuditActor actor;
}
