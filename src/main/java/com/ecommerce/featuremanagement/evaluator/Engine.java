package com.ecommerce.featuremanagement.evaluator;

import com.ecommerce.featuremanagement.cache.TieredCache;
import com.ecommerce.featuremanagement.config.Metrics;
import com.ecommerce.featuremanagement.domain.*;
import com.ecommerce.featuremanagement.repository.EnvironmentRepository;
import com.ecommerce.featuremanagement.repository.FlagRepository;
import com.ecommerce.featuremanagement.repository.SegmentRepository;
import com.fasterxml.jackson.databind.JsonNode;
import org.springframework.stereotype.Component;

import java.time.Instant;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import java.util.Map;

/** Core flag evaluation engine: mirrors the Go internal/evaluator/engine.go. */
@Component
public class Engine {

    private final TieredCache cache;
    private final FlagRepository flagRepo;
    private final SegmentRepository segmentRepo;
    private final EnvironmentRepository envRepo;
    private final Metrics metrics;

    public Engine(TieredCache cache, FlagRepository flagRepo, SegmentRepository segmentRepo,
                  EnvironmentRepository envRepo, Metrics metrics) {
        this.cache = cache;
        this.flagRepo = flagRepo;
        this.segmentRepo = segmentRepo;
        this.envRepo = envRepo;
        this.metrics = metrics;
    }

    public EvaluationResult evaluate(EvaluationContext ctx) {
        long startNanos = System.nanoTime();
        EvaluationResult result = doEvaluate(ctx);
        metrics.recordEvaluationDuration(ctx.flagKey, ctx.environmentId.toString(), System.nanoTime() - startNanos);
        String outcome = result.explanation.defaultServed ? "default" : "matched";
        metrics.incrementFlagEvaluation(ctx.flagKey, ctx.environmentId.toString(), outcome, result.cacheLayer);
        return result;
    }

    private EvaluationResult doEvaluate(EvaluationContext ctx) {
        Instant timestamp = ctx.timestamp != null ? ctx.timestamp : Instant.now();

        TieredCache.FlagLookup lookup = cache.getFlag(ctx.applicationId, ctx.environmentId, ctx.flagKey,
                () -> flagRepo.getByKey(ctx.applicationId, ctx.environmentId, ctx.flagKey));
        FeatureFlag flag = lookup.flag();
        String cacheLayer = lookup.cacheLayer();
        boolean cacheHit = !"db".equals(cacheLayer);

        String envSlug = "";
        try {
            envSlug = envRepo.getById(ctx.applicationId, ctx.environmentId).slug;
        } catch (Exception ignored) {
            // best-effort only; absence of an environment slug doesn't block evaluation
        }

        Explanation explanation = new Explanation();
        explanation.flagKey = flag.key;
        explanation.flagName = flag.name;
        explanation.flagVersion = flag.version;
        explanation.environmentSlug = envSlug;
        explanation.evaluatedAt = timestamp;
        explanation.entityId = ctx.entityId;
        explanation.entityType = ctx.entityType;

        if (flag.status != FlagStatus.ACTIVE) {
            ExplanationStep step = new ExplanationStep();
            step.outcome = "flag_disabled";
            explanation.steps.add(step);
            explanation.defaultServed = true;
            explanation.reason = flag.status == FlagStatus.ARCHIVED
                    ? "flag is archived; serving default value"
                    : "flag is inactive; serving default value";
            return buildResult(flag, flag.defaultValue, null, explanation, timestamp, cacheHit, cacheLayer);
        }

        List<Rule> rules = new ArrayList<>(flag.rules);
        rules.sort(Comparator.comparingInt(r -> r.priority));

        for (Rule rule : rules) {
            ExplanationStep step = new ExplanationStep();
            step.ruleId = rule.id;
            step.ruleName = rule.name;
            step.ruleType = rule.type;
            step.priority = rule.priority;

            if (!rule.enabled) {
                step.outcome = "disabled";
                explanation.steps.add(step);
                continue;
            }
            if ((rule.scheduleStart != null && timestamp.isBefore(rule.scheduleStart))
                    || (rule.scheduleEnd != null && timestamp.isAfter(rule.scheduleEnd))) {
                step.outcome = "schedule_miss";
                explanation.steps.add(step);
                continue;
            }

            boolean matched = matchRule(rule, ctx, step);

            if (!matched) {
                step.outcome = "skipped";
                explanation.steps.add(step);
                continue;
            }

            if (rule.rolloutPct != null) {
                int bucket = RolloutBucketer.computeBucket(flag.key, ctx.entityId, rule.rolloutSalt);
                step.rolloutBucket = bucket;
                step.rolloutPct = rule.rolloutPct;
                explanation.bucketValue = bucket;
                if (!RolloutBucketer.isInRollout(bucket, rule.rolloutPct)) {
                    step.outcome = "skipped";
                    explanation.steps.add(step);
                    continue;
                }
            }

            step.outcome = "matched";
            explanation.steps.add(step);

            VariantSelection selection = selectVariantValue(flag, rule, ctx);

            explanation.matchedRuleId = rule.id;
            explanation.matchedRuleName = rule.name;
            explanation.matchedRuleType = rule.type;
            explanation.defaultServed = false;
            explanation.reason = buildMatchReason(rule);

            return buildResult(flag, selection.value, selection.variantKey, explanation, timestamp, cacheHit, cacheLayer);
        }

        explanation.defaultServed = true;
        explanation.reason = "no rule matched; serving default value";
        return buildResult(flag, flag.defaultValue, null, explanation, timestamp, cacheHit, cacheLayer);
    }

    public List<EvaluationResult> batchEvaluate(BatchEvaluationRequest req) {
        List<String> keys = req.flagKeys;
        if (keys == null || keys.isEmpty()) {
            keys = new ArrayList<>();
            for (FeatureFlag flag : flagRepo.listByEnvironment(req.applicationId, req.environmentId)) {
                keys.add(flag.key);
            }
        }

        List<EvaluationResult> results = new ArrayList<>();
        for (String key : keys) {
            EvaluationContext ctx = new EvaluationContext();
            ctx.flagKey = key;
            ctx.applicationId = req.applicationId;
            ctx.environmentId = req.environmentId;
            ctx.entityId = req.entityId;
            ctx.entityType = req.entityType;
            ctx.attributes = req.attributes;
            ctx.timestamp = Instant.now();
            try {
                results.add(evaluate(ctx));
            } catch (Exception ignored) {
                // partial results: a single flag failure must not fail the whole batch
            }
        }
        return results;
    }

    private boolean matchRule(Rule rule, EvaluationContext ctx, ExplanationStep step) {
        switch (rule.type) {
            case TARGETING:
                populateConditionResults(step, rule.conditions, ctx.attributes);
                return TargetingMatcher.matchAllConditions(rule.conditions, ctx.attributes);
            case SEGMENT: {
                if (rule.segmentId == null) {
                    return false;
                }
                Segment segment = cache.getSegment(ctx.applicationId, rule.segmentId,
                        () -> segmentRepo.getById(rule.segmentId));
                step.segmentId = segment.id;
                step.segmentName = segment.name;
                return TargetingMatcher.matchSegment(segment, ctx.attributes);
            }
            case ROLLOUT:
            case SCHEDULE:
            default:
                if (rule.conditions != null && !rule.conditions.isEmpty()) {
                    populateConditionResults(step, rule.conditions, ctx.attributes);
                    return TargetingMatcher.matchAllConditions(rule.conditions, ctx.attributes);
                }
                return true;
        }
    }

    private void populateConditionResults(ExplanationStep step, List<Condition> conditions, Map<String, Object> attributes) {
        if (conditions == null) {
            return;
        }
        for (Condition c : conditions) {
            ExplanationMatchedCondition mc = new ExplanationMatchedCondition();
            mc.conditionId = c.id;
            mc.attribute = c.attribute;
            mc.operator = c.operator;
            mc.value = c.value;
            mc.actualValue = attributes == null ? null : attributes.get(c.attribute);
            mc.matched = ConditionMatcher.matches(c, attributes);
            step.conditionResults.add(mc);
        }
    }

    private static class VariantSelection {
        JsonNode value;
        String variantKey;
    }

    private VariantSelection selectVariantValue(FeatureFlag flag, Rule rule, EvaluationContext ctx) {
        VariantSelection selection = new VariantSelection();
        if (rule.variants == null || rule.variants.isEmpty()) {
            if (flag.variants != null && !flag.variants.isEmpty()) {
                Variant v = flag.variants.get(0);
                selection.value = v.value;
                selection.variantKey = v.key;
            } else {
                selection.value = flag.defaultValue;
            }
            return selection;
        }

        int bucket = RolloutBucketer.computeBucket(flag.key, ctx.entityId, rule.rolloutSalt);
        for (VariantAllocation alloc : rule.variants) {
            if (bucket >= alloc.rolloutFrom && bucket < alloc.rolloutTo) {
                for (Variant v : flag.variants) {
                    if (v.id.equals(alloc.variantId)) {
                        selection.value = v.value;
                        selection.variantKey = v.key;
                        return selection;
                    }
                }
            }
        }
        selection.value = flag.defaultValue;
        return selection;
    }

    private String buildMatchReason(Rule rule) {
        switch (rule.type) {
            case TARGETING:
                return "matched targeting rule \"" + rule.name + "\"";
            case SEGMENT:
                return "matched segment rule \"" + rule.name + "\"";
            case ROLLOUT:
                return "matched rollout rule \"" + rule.name + "\" (" + rule.rolloutPct + "% rollout)";
            case SCHEDULE:
                return "matched schedule rule \"" + rule.name + "\"";
            default:
                return "matched rule \"" + rule.name + "\"";
        }
    }

    private EvaluationResult buildResult(FeatureFlag flag, JsonNode value, String variantKey, Explanation explanation,
                                          Instant timestamp, boolean cacheHit, String cacheLayer) {
        EvaluationResult result = new EvaluationResult();
        result.flagKey = flag.key;
        result.flagType = flag.type;
        result.value = value;
        result.variantKey = variantKey;
        result.enabled = isEnabled(flag, value);
        result.explanation = explanation;
        result.evaluatedAt = timestamp;
        result.cacheHit = cacheHit;
        result.cacheLayer = cacheLayer;
        return result;
    }

    private boolean isEnabled(FeatureFlag flag, JsonNode value) {
        if (flag.type == FlagType.BOOLEAN) {
            return value != null && value.isBoolean() && value.asBoolean();
        }
        return flag.status == FlagStatus.ACTIVE;
    }
}
