package com.ecommerce.featuremanagement.service.dto;

import com.ecommerce.featuremanagement.domain.AuditActor;
import java.util.UUID;

public class CreateEnvironmentInput {
    public UUID applicationId;
    public String name;
    public String slug;
    public boolean requiresApproval;
    public AuditActor actor;
}
