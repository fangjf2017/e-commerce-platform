package com.ecommerce.featuremanagement.cache;

import com.ecommerce.featuremanagement.util.JsonUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.data.redis.connection.MessageListener;
import org.springframework.data.redis.listener.PatternTopic;
import org.springframework.data.redis.listener.RedisMessageListenerContainer;
import org.springframework.stereotype.Component;

import jakarta.annotation.PostConstruct;
import java.util.UUID;

/**
 * Subscribes to Redis pub/sub invalidation channels and evicts the corresponding
 * L1/L2 cache entries on every node. Mirrors the Go cache.Invalidator.
 */
@Component
public class Invalidator implements MessageListener {

    private static final Logger log = LoggerFactory.getLogger(Invalidator.class);

    private final RedisMessageListenerContainer container;
    private final L2Cache l2;
    private final TieredCache tieredCache;

    public Invalidator(RedisMessageListenerContainer container, L2Cache l2, TieredCache tieredCache) {
        this.container = container;
        this.l2 = l2;
        this.tieredCache = tieredCache;
    }

    @PostConstruct
    public void subscribe() {
        container.addMessageListener(this, new PatternTopic(CacheKeys.FLAG_INVALIDATION_PATTERN));
        container.addMessageListener(this, new PatternTopic(CacheKeys.SEGMENT_INVALIDATION_PATTERN));
    }

    @Override
    public void onMessage(org.springframework.data.redis.connection.Message message, byte[] pattern) {
        String channel = new String(message.getChannel());
        try {
            InvalidationMessage msg = JsonUtils.MAPPER.readValue(message.getBody(), InvalidationMessage.class);
            if (channel.startsWith("ff:invalidate:")) {
                tieredCache.deleteFlag(UUID.fromString(msg.appId), UUID.fromString(msg.envId), msg.flagKey);
                log.info("invalidated flag cache key={} action={} version={}", msg.flagKey, msg.action, msg.version);
            } else if (channel.startsWith("seg:invalidate:")) {
                tieredCache.deleteSegment(UUID.fromString(msg.appId), UUID.fromString(msg.segmentId));
                log.info("invalidated segment cache id={}", msg.segmentId);
            }
        } catch (Exception e) {
            log.warn("failed to process invalidation message on channel {}: {}", channel, e.getMessage());
        }
    }

    public void publishFlagInvalidation(UUID appId, UUID envId, String flagKey, String action, long version) {
        InvalidationMessage msg = new InvalidationMessage();
        msg.flagKey = flagKey;
        msg.version = version;
        msg.action = action;
        msg.appId = appId.toString();
        msg.envId = envId.toString();
        l2.publishInvalidation(CacheKeys.flagInvalidationChannel(appId, envId), JsonUtils.toJson(msg));
    }

    public void publishSegmentInvalidation(UUID appId, UUID segId) {
        InvalidationMessage msg = new InvalidationMessage();
        msg.appId = appId.toString();
        msg.segmentId = segId.toString();
        l2.publishInvalidation(CacheKeys.segmentInvalidationChannel(appId), JsonUtils.toJson(msg));
    }
}
