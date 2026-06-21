package com.ecommerce.featuremanagement.cache;

import com.ecommerce.featuremanagement.domain.FeatureFlag;
import com.ecommerce.featuremanagement.domain.Segment;
import com.ecommerce.featuremanagement.util.JsonUtils;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Component;

import java.time.Duration;

/** L2 cache backed by Redis, the Java equivalent of the Go go-redis layer. */
@Component
public class L2Cache {

    private static final Duration FLAG_TTL = Duration.ofMinutes(5);
    private static final Duration SEGMENT_TTL = Duration.ofMinutes(10);

    private final StringRedisTemplate redis;

    public L2Cache(StringRedisTemplate redis) {
        this.redis = redis;
    }

    public StringRedisTemplate client() {
        return redis;
    }

    public FeatureFlag getFlag(String key) {
        String raw = redis.opsForValue().get(key);
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
        redis.opsForValue().set(key, JsonUtils.toJson(flag), FLAG_TTL);
    }

    public void deleteFlag(String key) {
        redis.delete(key);
    }

    public Segment getSegment(String key) {
        String raw = redis.opsForValue().get(key);
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
        redis.opsForValue().set(key, JsonUtils.toJson(segment), SEGMENT_TTL);
    }

    public void deleteSegment(String key) {
        redis.delete(key);
    }

    public void publishInvalidation(String channel, String payload) {
        redis.convertAndSend(channel, payload);
    }
}
