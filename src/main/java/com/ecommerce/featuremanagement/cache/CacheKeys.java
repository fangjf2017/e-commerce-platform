package com.ecommerce.featuremanagement.cache;

import java.util.UUID;

public final class CacheKeys {

    private CacheKeys() {}

    public static String flagKey(UUID appId, UUID envId, String flagKey) {
        return "ff:" + appId + ":" + envId + ":" + flagKey;
    }

    public static String segmentKey(UUID appId, UUID segId) {
        return "seg:" + appId + ":" + segId;
    }

    public static String flagInvalidationChannel(UUID appId, UUID envId) {
        return "ff:invalidate:" + appId + ":" + envId;
    }

    public static String segmentInvalidationChannel(UUID appId) {
        return "seg:invalidate:" + appId;
    }

    public static final String FLAG_INVALIDATION_PATTERN = "ff:invalidate:*";
    public static final String SEGMENT_INVALIDATION_PATTERN = "seg:invalidate:*";
}
