package com.ecommerce.featuremanagement.service.dto;

import java.time.Instant;
import java.util.UUID;

public class ListAuditInput {
    public UUID applicationId;
    public String resourceType;
    public String action;
    public String actorId;
    public Instant after;
    public Instant before;
    public int limit;
    public int offset;
}
