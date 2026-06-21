package com.ecommerce.featuremanagement.cache;

import com.ecommerce.featuremanagement.domain.FeatureFlag;
import com.ecommerce.featuremanagement.domain.Segment;
import com.ecommerce.featuremanagement.util.JsonUtils;
import com.github.benmanes.caffeine.cache.Cache;
import com.github.benmanes.caffeine.cache.Caffeine;
import org.springframework.stereotype.Component;

import java.time.Duration;

/** In-process L1 cache (Caffeine), the Java equivalent of the Go Ristretto layer. */
@Component
public class L1Cache {

    private static final Duration TTL = Duration.ofSeconds(30);

    private final Cache<String, byte[]> cache = Caffeine.newBuilder()
            .maximumWeight(256L * 1024 * 1024)
            .weigher((String k, byte[] v) -> v.length)
            .expireAfterWrite(TTL)
            .build();

    public FeatureFlag getFlag(String key) {
        byte[] raw = cache.getIfPresent(key);
        if (raw == null) {
            return null;
        }
        try {
            return JsonUtils.MAPPER.readValue(raw, FeatureFlag.class);
        } catch (Exception e) {
            return null;
        }
    }

    public void setFlag(String key, FeatureFlag flag) {
        cache.put(key, JsonUtils.toJson(flag).getBytes());
    }

    public void deleteFlag(String key) {
        cache.invalidate(key);
    }

    public Segment getSegment(String key) {
        byte[] raw = cache.getIfPresent(key);
        if (raw == null) {
            return null;
        }
        try {
            return JsonUtils.MAPPER.readValue(raw, Segment.class);
        } catch (Exception e) {
            return null;
        }
    }

    public void setSegment(String key, Segment segment) {
        cache.put(key, JsonUtils.toJson(segment).getBytes());
    }

    public void deleteSegment(String key) {
        cache.invalidate(key);
    }
}
