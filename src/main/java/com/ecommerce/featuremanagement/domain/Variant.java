package com.ecommerce.featuremanagement.domain;

import com.fasterxml.jackson.databind.JsonNode;
import java.util.UUID;

public class Variant {
    public UUID id;
    public UUID flagId;
    public String key;
    public JsonNode value;
    public String description;
}
