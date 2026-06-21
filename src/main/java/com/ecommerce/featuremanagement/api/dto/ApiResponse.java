package com.ecommerce.featuremanagement.api.dto;

public class ApiResponse<T> {
    public T data;
    public ApiMeta meta;

    public static <T> ApiResponse<T> of(T data) {
        ApiResponse<T> resp = new ApiResponse<>();
        resp.data = data;
        return resp;
    }

    public static <T> ApiResponse<T> of(T data, ApiMeta meta) {
        ApiResponse<T> resp = new ApiResponse<>();
        resp.data = data;
        resp.meta = meta;
        return resp;
    }
}
