package com.ecommerce.featuremanagement.domain;

import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.UUID;

public class Segment {
    public UUID id;
    public UUID applicationId;
    public String name;
    public String description;
    public SegmentOperator operator = SegmentOperator.ALL;
    public List<SegmentRule> rules = new ArrayList<>();
    public Instant createdAt;
    public Instant updatedAt;
    public long version;
}
