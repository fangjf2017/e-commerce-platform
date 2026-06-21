package com.ecommerce.featuremanagement.service;

import com.ecommerce.featuremanagement.cache.Invalidator;
import com.ecommerce.featuremanagement.cache.TieredCache;
import com.ecommerce.featuremanagement.domain.*;
import com.ecommerce.featuremanagement.repository.AuditRepository;
import com.ecommerce.featuremanagement.repository.FlagRepository;
import com.ecommerce.featuremanagement.repository.PagedResult;
import com.ecommerce.featuremanagement.repository.RuleRepository;
import com.ecommerce.featuremanagement.service.dto.*;
import com.ecommerce.featuremanagement.util.JsonUtils;
import org.springframework.stereotype.Service;

import java.time.Instant;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.UUID;

@Service
public class FlagService {

    private final FlagRepository flags;
    private final RuleRepository rules;
    private final AuditRepository audit;
    private final TieredCache cache;
    private final Invalidator invalidator;

    public FlagService(FlagRepository flags, RuleRepository rules, AuditRepository audit,
                        TieredCache cache, Invalidator invalidator) {
        this.flags = flags;
        this.rules = rules;
        this.audit = audit;
        this.cache = cache;
        this.invalidator = invalidator;
    }

    public FeatureFlag create(CreateFlagInput inp) {
        FeatureFlag flag = new FeatureFlag();
        flag.applicationId = inp.applicationId;
        flag.environmentId = inp.environmentId;
        flag.key = inp.key;
        flag.name = inp.name;
        flag.description = inp.description != null ? inp.description : "";
        flag.type = inp.type;
        flag.status = FlagStatus.INACTIVE;
        flag.defaultValue = inp.defaultValue;
        flag.tags = inp.tags != null ? inp.tags : new ArrayList<>();

        flags.create(flag);

        AuditEvent event = newAuditEvent(flag, AuditAction.FLAG_CREATED, inp.actor);
        event.after = JsonUtils.parse(JsonUtils.toJson(flag));
        audit.create(event);

        invalidator.publishFlagInvalidation(flag.applicationId, flag.environmentId, flag.key, "flag.created", flag.version);
        return flag;
    }

    public FeatureFlag get(UUID appId, UUID envId, String key) {
        return flags.getByKey(appId, envId, key);
    }

    public FeatureFlag getById(UUID id) {
        return flags.getById(id);
    }

    public PagedResult<FeatureFlag> list(UUID appId, UUID envId, FlagStatus status, List<String> tags, int limit, int offset) {
        if (limit <= 0) {
            limit = 20;
        }
        return flags.list(appId, envId, status, tags, limit, offset);
    }

    public FeatureFlag update(UUID appId, UUID envId, String key, UpdateFlagInput inp) {
        FeatureFlag flag = flags.getByKey(appId, envId, key);
        String before = JsonUtils.toJson(flag);

        flag.name = inp.name;
        flag.description = inp.description != null ? inp.description : "";
        flag.defaultValue = inp.defaultValue;
        flag.tags = inp.tags != null ? inp.tags : flag.tags;

        flags.update(flag);

        AuditEvent event = newAuditEvent(flag, AuditAction.FLAG_UPDATED, inp.actor);
        event.before = JsonUtils.parse(before);
        event.after = JsonUtils.parse(JsonUtils.toJson(flag));
        audit.create(event);

        invalidator.publishFlagInvalidation(flag.applicationId, flag.environmentId, flag.key, "flag.updated", flag.version);
        return flag;
    }

    public void delete(UUID appId, UUID envId, String key, AuditActor actor) {
        FeatureFlag flag = flags.getByKey(appId, envId, key);
        String before = JsonUtils.toJson(flag);

        flags.delete(flag.id);

        AuditEvent event = newAuditEvent(flag, AuditAction.FLAG_DELETED, actor);
        event.before = JsonUtils.parse(before);
        audit.create(event);

        invalidator.publishFlagInvalidation(flag.applicationId, flag.environmentId, flag.key, "flag.deleted", flag.version);
    }

    public FeatureFlag setStatus(UUID appId, UUID envId, String key, FlagStatus status, AuditActor actor) {
        FeatureFlag flag = flags.getByKey(appId, envId, key);
        String before = JsonUtils.toJson(flag);

        flags.updateStatus(flag.id, status);
        flag.status = status;

        AuditEvent event = newAuditEvent(flag, AuditAction.FLAG_STATUS_CHANGED, actor);
        event.before = JsonUtils.parse(before);
        event.after = JsonUtils.parse(JsonUtils.toJson(flag));
        event.metadata = new HashMap<>();
        event.metadata.put("status", status.value());
        audit.create(event);

        invalidator.publishFlagInvalidation(flag.applicationId, flag.environmentId, flag.key, "flag.status_changed", flag.version);
        return flag;
    }

    public Rule createRule(CreateRuleInput inp) {
        FeatureFlag flag = flags.getById(inp.flagId);

        Rule rule = new Rule();
        rule.flagId = inp.flagId;
        rule.type = inp.type;
        rule.priority = inp.priority;
        rule.name = inp.name;
        rule.description = inp.description != null ? inp.description : "";
        rule.conditions = toConditions(inp.conditions);
        rule.segmentId = inp.segmentId;
        rule.rolloutPct = inp.rolloutPct;
        rule.rolloutSalt = UUID.randomUUID();
        rule.variants = toAllocations(inp.variants);
        rule.scheduleStart = inp.scheduleStart;
        rule.scheduleEnd = inp.scheduleEnd;
        rule.enabled = inp.enabled;

        rules.create(rule);

        AuditEvent event = newAuditEvent(flag, AuditAction.RULE_CREATED, inp.actor);
        event.resourceType = "rule";
        event.resourceId = rule.id;
        event.resourceKey = rule.name;
        event.after = JsonUtils.parse(JsonUtils.toJson(rule));
        audit.create(event);

        invalidator.publishFlagInvalidation(flag.applicationId, flag.environmentId, flag.key, "flag.updated", flag.version);
        return rule;
    }

    public Rule updateRule(UUID ruleId, UpdateRuleInput inp) {
        Rule rule = rules.getById(ruleId);
        FeatureFlag flag = flags.getById(rule.flagId);
        String before = JsonUtils.toJson(rule);

        rule.type = inp.type;
        rule.priority = inp.priority;
        rule.name = inp.name;
        rule.description = inp.description != null ? inp.description : "";
        rule.conditions = toConditions(inp.conditions);
        rule.segmentId = inp.segmentId;
        rule.rolloutPct = inp.rolloutPct;
        rule.variants = toAllocations(inp.variants);
        rule.scheduleStart = inp.scheduleStart;
        rule.scheduleEnd = inp.scheduleEnd;
        rule.enabled = inp.enabled;

        rules.update(rule);

        AuditEvent event = newAuditEvent(flag, AuditAction.RULE_UPDATED, inp.actor);
        event.resourceType = "rule";
        event.resourceId = rule.id;
        event.resourceKey = rule.name;
        event.before = JsonUtils.parse(before);
        event.after = JsonUtils.parse(JsonUtils.toJson(rule));
        audit.create(event);

        invalidator.publishFlagInvalidation(flag.applicationId, flag.environmentId, flag.key, "flag.updated", flag.version);
        return rule;
    }

    public void deleteRule(UUID ruleId, AuditActor actor) {
        Rule rule = rules.getById(ruleId);
        FeatureFlag flag = flags.getById(rule.flagId);

        rules.delete(ruleId);

        AuditEvent event = newAuditEvent(flag, AuditAction.RULE_DELETED, actor);
        event.resourceType = "rule";
        event.resourceId = ruleId;
        event.resourceKey = rule.name;
        audit.create(event);

        invalidator.publishFlagInvalidation(flag.applicationId, flag.environmentId, flag.key, "flag.updated", flag.version);
    }

    public void reorderRules(UUID flagId, List<UUID> orderedRuleIds) {
        FeatureFlag flag = flags.getById(flagId);
        rules.reorderByPriority(orderedRuleIds);
        invalidator.publishFlagInvalidation(flag.applicationId, flag.environmentId, flag.key, "flag.updated", flag.version);
    }

    private AuditEvent newAuditEvent(FeatureFlag flag, AuditAction action, AuditActor actor) {
        AuditEvent event = new AuditEvent();
        event.applicationId = flag.applicationId;
        event.environmentId = flag.environmentId;
        event.action = action;
        event.resourceType = "feature_flag";
        event.resourceId = flag.id;
        event.resourceKey = flag.key;
        event.actor = actor;
        event.occurredAt = Instant.now();
        return event;
    }

    private List<Condition> toConditions(List<ConditionInput> inputs) {
        List<Condition> out = new ArrayList<>();
        if (inputs == null) {
            return out;
        }
        for (ConditionInput i : inputs) {
            Condition c = new Condition();
            c.attribute = i.attribute;
            c.operator = i.operator;
            c.value = i.value;
            c.negate = i.negate;
            out.add(c);
        }
        return out;
    }

    private List<VariantAllocation> toAllocations(List<VariantAllocationInput> inputs) {
        List<VariantAllocation> out = new ArrayList<>();
        if (inputs == null) {
            return out;
        }
        for (VariantAllocationInput i : inputs) {
            VariantAllocation a = new VariantAllocation();
            a.variantId = i.variantId;
            a.rolloutFrom = i.rolloutFrom;
            a.rolloutTo = i.rolloutTo;
            out.add(a);
        }
        return out;
    }
}
