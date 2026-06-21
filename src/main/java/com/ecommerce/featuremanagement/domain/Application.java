package com.ecommerce.featuremanagement.domain;

import java.time.Instant;
import java.util.UUID;

public class Application {
    public UUID id;
    public String name;
    public String slug;
    public String description;
    public Instant createdAt;
    public Instant updatedAt;
}
