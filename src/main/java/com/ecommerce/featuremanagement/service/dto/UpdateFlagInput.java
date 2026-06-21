package com.ecommerce.featuremanagement.service.dto;

import com.ecommerce.featuremanagement.domain.AuditActor;
import com.fasterxml.jackson.databind.JsonNode;

import java.util.List;

public class UpdateFlagInput {
    public String name;
    public String description;
    public JsonNode defaultValue;
    public List<String> tags;
    public AuditActor actor;
}
