package com.ecommerce.featuremanagement.api.dto;

public class ApiMeta {
    public int total;
    public int limit;
    public int offset;

    public ApiMeta() {}

    public ApiMeta(int total, int limit, int offset) {
        this.total = total;
        this.limit = limit;
        this.offset = offset;
    }
}
