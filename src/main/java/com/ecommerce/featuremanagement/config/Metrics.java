package com.ecommerce.featuremanagement.config;

import io.micrometer.core.instrument.Counter;
import io.micrometer.core.instrument.MeterRegistry;
import io.micrometer.core.instrument.Timer;
import org.springframework.stereotype.Component;

import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.TimeUnit;

/** Central registry of application-specific metrics, namespaced like the Go "fms_*" series. */
@Component
public class Metrics {

    private final MeterRegistry registry;
    private final ConcurrentHashMap<String, Counter> counters = new ConcurrentHashMap<>();
    private final ConcurrentHashMap<String, Timer> timers = new ConcurrentHashMap<>();

    public Metrics(MeterRegistry registry) {
        this.registry = registry;
    }

    public void incrementCacheHit(String tier) {
        counter("fms_cache_hits_total", "tier", tier).increment();
    }

    public void incrementCacheMiss(String tier) {
        counter("fms_cache_misses_total", "tier", tier).increment();
    }

    public void incrementCacheInvalidation(String reason) {
        counter("fms_cache_invalidations_total", "reason", reason).increment();
    }

    public void incrementFlagEvaluation(String flagKey, String environment, String result, String cacheLayer) {
        registry.counter("fms_flag_evaluations_total", "flag_key", flagKey, "environment", environment,
                "result", result, "cache_layer", cacheLayer).increment();
    }

    public void recordEvaluationDuration(String flagKey, String environment, long nanos) {
        registry.timer("fms_evaluation_duration_seconds", "flag_key", flagKey, "environment", environment)
                .record(nanos, TimeUnit.NANOSECONDS);
    }

    public void recordHttpRequest(String method, String path, int statusCode, long durationNanos) {
        registry.counter("fms_http_requests_total", "method", method, "path", path,
                "status_code", String.valueOf(statusCode)).increment();
        registry.timer("fms_http_request_duration_seconds", "method", method, "path", path)
                .record(durationNanos, TimeUnit.NANOSECONDS);
    }

    private Counter counter(String name, String tagKey, String tagValue) {
        return counters.computeIfAbsent(name + ":" + tagKey + "=" + tagValue,
                k -> registry.counter(name, tagKey, tagValue));
    }
}
