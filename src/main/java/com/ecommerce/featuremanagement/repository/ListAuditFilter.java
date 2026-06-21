package com.ecommerce.featuremanagement.repository;

import java.time.Instant;

public class ListAuditFilter {
    public String resourceType;
    public String action;
    public String actorId;
    public Instant after;
    public Instant before;
    public int limit = 50;
    public int offset = 0;
}
