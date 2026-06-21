package com.ecommerce.featuremanagement.cache;

public class InvalidationMessage {
    public String flagKey;
    public long version;
    public String action;
    public String appId;
    public String envId;
    public String segmentId;
}
