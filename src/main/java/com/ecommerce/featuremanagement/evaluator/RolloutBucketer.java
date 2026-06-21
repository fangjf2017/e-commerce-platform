package com.ecommerce.featuremanagement.evaluator;

import java.util.UUID;

/** Deterministic, sticky bucketing for percentage rollouts, mirroring the Go evaluator/rollout.go. */
public final class RolloutBucketer {

    private static final int BUCKET_SPACE = 10000;

    private RolloutBucketer() {}

    public static int computeBucket(String flagKey, String entityId, UUID salt) {
        String input = flagKey + ":" + entityId + ":" + salt.toString();
        long hash = Murmur3.sum32(input);
        return (int) (hash % BUCKET_SPACE);
    }

    public static boolean isInRollout(int bucket, int rolloutPct) {
        if (rolloutPct <= 0) {
            return false;
        }
        if (rolloutPct >= 100) {
            return true;
        }
        return bucket < rolloutPct * 100;
    }
}
