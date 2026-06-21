package com.ecommerce.featuremanagement.repository;

import java.util.List;

public class PagedResult<T> {
    public final List<T> items;
    public final int total;

    public PagedResult(List<T> items, int total) {
        this.items = items;
        this.total = total;
    }
}
