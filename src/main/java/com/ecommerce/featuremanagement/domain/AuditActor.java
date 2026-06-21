package com.ecommerce.featuremanagement.domain;

public class AuditActor {
    public String id;
    public String email;
    public String type; // user | api_key | system

    public AuditActor() {}

    public AuditActor(String id, String email, String type) {
        this.id = id;
        this.email = email;
        this.type = type;
    }
}
