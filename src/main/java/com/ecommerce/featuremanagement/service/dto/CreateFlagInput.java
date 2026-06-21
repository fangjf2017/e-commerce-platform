package com.ecommerce.featuremanagement.service.dto;

import com.ecommerce.featuremanagement.domain.AuditActor;
import com.ecommerce.featuremanagement.domain.FlagType;
import com.fasterxml.jackson.databind.JsonNode;

import java.util.List;
import java.util.UUID;

public class CreateFlagInput {
    public UUID applicationId;
    public UUID environmentId;
    public String key;
    public String name;
    public String description;
    public FlagType type;
    public JsonNode defaultValue;
    public List<String> tags;
    public AuditActor actor;
}
