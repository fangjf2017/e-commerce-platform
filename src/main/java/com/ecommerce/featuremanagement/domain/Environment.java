package com.ecommerce.featuremanagement.domain;

import java.time.Instant;
import java.util.UUID;

public class Environment {
    public UUID id;
    public UUID applicationId;
    public String name;
    public String slug;
    public boolean requiresApproval;
    public Instant createdAt;
    public Instant updatedAt;
}
