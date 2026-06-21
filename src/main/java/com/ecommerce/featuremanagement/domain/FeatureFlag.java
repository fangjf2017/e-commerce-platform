package com.ecommerce.featuremanagement.domain;

import com.fasterxml.jackson.databind.JsonNode;
import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.UUID;

public class FeatureFlag {
    public UUID id;
    public UUID applicationId;
    public UUID environmentId;
    public String key;
    public String name;
    public String description;
    public FlagType type;
    public FlagStatus status = FlagStatus.INACTIVE;
    public JsonNode defaultValue;
    public List<String> tags = new ArrayList<>();
    public List<Rule> rules = new ArrayList<>();
    public List<Variant> variants = new ArrayList<>();
    public Instant createdAt;
    public Instant updatedAt;
    public long version;
}
