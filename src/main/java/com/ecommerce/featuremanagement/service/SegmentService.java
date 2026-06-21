package com.ecommerce.featuremanagement.service;

import com.ecommerce.featuremanagement.cache.Invalidator;
import com.ecommerce.featuremanagement.domain.*;
import com.ecommerce.featuremanagement.repository.AuditRepository;
import com.ecommerce.featuremanagement.repository.PagedResult;
import com.ecommerce.featuremanagement.repository.SegmentRepository;
import com.ecommerce.featuremanagement.service.dto.CreateSegmentInput;
import com.ecommerce.featuremanagement.service.dto.SegmentRuleInput;
import com.ecommerce.featuremanagement.service.dto.UpdateSegmentInput;
import com.ecommerce.featuremanagement.util.JsonUtils;
import org.springframework.stereotype.Service;

import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.UUID;

@Service
public class SegmentService {

    private final SegmentRepository segments;
    private final AuditRepository audit;
    private final Invalidator invalidator;

    public SegmentService(SegmentRepository segments, AuditRepository audit, Invalidator invalidator) {
        this.segments = segments;
        this.audit = audit;
        this.invalidator = invalidator;
    }

    public Segment create(CreateSegmentInput inp) {
        Segment seg = new Segment();
        seg.applicationId = inp.applicationId;
        seg.name = inp.name;
        seg.description = inp.description != null ? inp.description : "";
        seg.operator = inp.operator != null ? inp.operator : SegmentOperator.ALL;
        seg.rules = toRules(inp.rules);

        segments.create(seg);

        AuditEvent event = new AuditEvent();
        event.applicationId = seg.applicationId;
        event.action = AuditAction.SEGMENT_CREATED;
        event.resourceType = "segment";
        event.resourceId = seg.id;
        event.resourceKey = seg.name;
        event.actor = inp.actor;
        event.after = JsonUtils.parse(JsonUtils.toJson(seg));
        event.occurredAt = Instant.now();
        audit.create(event);

        invalidator.publishSegmentInvalidation(seg.applicationId, seg.id);
        return seg;
    }

    public Segment get(UUID segId) {
        return segments.getById(segId);
    }

    public PagedResult<Segment> list(UUID applicationId, int limit, int offset) {
        List<Segment> all = segments.list(applicationId);
        int total = all.size();
        if (limit <= 0) {
            limit = 20;
        }
        if (offset < 0) {
            offset = 0;
        }
        if (offset >= total) {
            return new PagedResult<>(List.of(), total);
        }
        int end = Math.min(offset + limit, total);
        return new PagedResult<>(all.subList(offset, end), total);
    }

    public Segment update(UUID segId, UpdateSegmentInput inp) {
        Segment existing = segments.getById(segId);
        String before = JsonUtils.toJson(existing);

        existing.name = inp.name;
        existing.description = inp.description != null ? inp.description : "";
        existing.operator = inp.operator != null ? inp.operator : SegmentOperator.ALL;
        existing.rules = toRules(inp.rules);

        segments.update(existing);

        AuditEvent event = new AuditEvent();
        event.applicationId = existing.applicationId;
        event.action = AuditAction.SEGMENT_UPDATED;
        event.resourceType = "segment";
        event.resourceId = existing.id;
        event.resourceKey = existing.name;
        event.actor = inp.actor;
        event.before = JsonUtils.parse(before);
        event.after = JsonUtils.parse(JsonUtils.toJson(existing));
        event.occurredAt = Instant.now();
        audit.create(event);

        invalidator.publishSegmentInvalidation(existing.applicationId, existing.id);
        return existing;
    }

    public void delete(UUID id, AuditActor actor) {
        Segment existing = segments.getById(id);
        segments.delete(id);

        AuditEvent event = new AuditEvent();
        event.applicationId = existing.applicationId;
        event.action = AuditAction.SEGMENT_DELETED;
        event.resourceType = "segment";
        event.resourceId = id;
        event.resourceKey = id.toString();
        event.actor = actor;
        event.occurredAt = Instant.now();
        audit.create(event);

        invalidator.publishSegmentInvalidation(existing.applicationId, id);
    }

    private List<SegmentRule> toRules(List<SegmentRuleInput> inputs) {
        List<SegmentRule> out = new ArrayList<>();
        if (inputs == null) {
            return out;
        }
        for (SegmentRuleInput i : inputs) {
            SegmentRule r = new SegmentRule();
            r.attribute = i.attribute;
            r.operator = i.operator;
            r.value = i.value;
            out.add(r);
        }
        return out;
    }
}
