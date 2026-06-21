package com.ecommerce.featuremanagement.api.dto;

public class ErrorResponse {
    public ErrorBody error;

    public ErrorResponse(String code, String message) {
        this.error = new ErrorBody(code, message);
    }

    public static class ErrorBody {
        public String code;
        public String message;

        public ErrorBody(String code, String message) {
            this.code = code;
            this.message = message;
        }
    }
}
