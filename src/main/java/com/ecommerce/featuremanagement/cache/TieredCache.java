package com.ecommerce.featuremanagement.cache;

import com.ecommerce.featuremanagement.config.Metrics;
import com.ecommerce.featuremanagement.domain.FeatureFlag;
import com.ecommerce.featuremanagement.domain.Segment;
import org.springframework.stereotype.Component;

import java.util.UUID;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ConcurrentHashMap;
import java.util.function.Supplier;

/**
 * L1 (Caffeine) -> L2 (Redis) -> loader lookup chain, with in-flight request
 * de-duplication per key (the Java analogue of Go's singleflight.Group).
 */
@Component
public class TieredCache {

    private final L1Cache l1;
    private final L2Cache l2;
    private final Metrics metrics;

    private final ConcurrentHashMap<String, CompletableFuture<FlagLookup>> flagFlight = new ConcurrentHashMap<>();
    private final ConcurrentHashMap<String, CompletableFuture<Segment>> segmentFlight = new ConcurrentHashMap<>();

    public TieredCache(L1Cache l1, L2Cache l2, Metrics metrics) {
        this.l1 = l1;
        this.l2 = l2;
        this.metrics = metrics;
    }

    public record FlagLookup(FeatureFlag flag, String cacheLayer) {}

    public FlagLookup getFlag(UUID appId, UUID envId, String flagKey, Supplier<FeatureFlag> loader) {
        String key = CacheKeys.flagKey(appId, envId, flagKey);

        FeatureFlag fromL1 = l1.getFlag(key);
        if (fromL1 != null) {
            metrics.incrementCacheHit("l1");
            return new FlagLookup(fromL1, "l1");
        }
        metrics.incrementCacheMiss("l1");

        FeatureFlag fromL2 = l2.getFlag(key);
        if (fromL2 != null) {
            metrics.incrementCacheHit("l2");
            l1.setFlag(key, fromL2);
            return new FlagLookup(fromL2, "l2");
        }
        metrics.incrementCacheMiss("l2");

        CompletableFuture<FlagLookup> future = flagFlight.computeIfAbsent(key, k -> CompletableFuture.supplyAsync(() -> {
            FeatureFlag loaded = loader.get();
            l2.setFlag(key, loaded);
            l1.setFlag(key, loaded);
            return new FlagLookup(loaded, "db");
        }));
        try {
            return future.join();
        } finally {
            flagFlight.remove(key, future);
        }
    }

    public Segment getSegment(UUID appId, UUID segId, Supplier<Segment> loader) {
        String key = CacheKeys.segmentKey(appId, segId);

        Segment fromL1 = l1.getSegment(key);
        if (fromL1 != null) {
            metrics.incrementCacheHit("l1");
            return fromL1;
        }
        metrics.incrementCacheMiss("l1");

        Segment fromL2 = l2.getSegment(key);
        if (fromL2 != null) {
            metrics.incrementCacheHit("l2");
            l1.setSegment(key, fromL2);
            return fromL2;
        }
        metrics.incrementCacheMiss("l2");

        CompletableFuture<Segment> future = segmentFlight.computeIfAbsent(key, k -> CompletableFuture.supplyAsync(() -> {
            Segment loaded = loader.get();
            l2.setSegment(key, loaded);
            l1.setSegment(key, loaded);
            return loaded;
        }));
        try {
            return future.join();
        } finally {
            segmentFlight.remove(key, future);
        }
    }

    public void setFlag(FeatureFlag flag) {
        String key = CacheKeys.flagKey(flag.applicationId, flag.environmentId, flag.key);
        l2.setFlag(key, flag);
        l1.setFlag(key, flag);
    }

    public void deleteFlag(UUID appId, UUID envId, String flagKey) {
        String key = CacheKeys.flagKey(appId, envId, flagKey);
        l1.deleteFlag(key);
        l2.deleteFlag(key);
        metrics.incrementCacheInvalidation("flag_delete");
    }

    public void setSegment(Segment segment) {
        String key = CacheKeys.segmentKey(segment.applicationId, segment.id);
        l2.setSegment(key, segment);
        l1.setSegment(key, segment);
    }

    public void deleteSegment(UUID appId, UUID segId) {
        String key = CacheKeys.segmentKey(appId, segId);
        l1.deleteSegment(key);
        l2.deleteSegment(key);
        metrics.incrementCacheInvalidation("segment_delete");
    }
}
