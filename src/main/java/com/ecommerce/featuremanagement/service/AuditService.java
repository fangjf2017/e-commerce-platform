package com.ecommerce.featuremanagement.service;

import com.ecommerce.featuremanagement.domain.AuditEvent;
import com.ecommerce.featuremanagement.repository.AuditRepository;
import com.ecommerce.featuremanagement.repository.ListAuditFilter;
import com.ecommerce.featuremanagement.repository.PagedResult;
import com.ecommerce.featuremanagement.service.dto.ListAuditInput;
import org.springframework.stereotype.Service;

import java.util.UUID;

@Service
public class AuditService {

    private final AuditRepository audit;

    public AuditService(AuditRepository audit) {
        this.audit = audit;
    }

    public PagedResult<AuditEvent> list(ListAuditInput inp) {
        ListAuditFilter filter = new ListAuditFilter();
        filter.resourceType = inp.resourceType;
        filter.action = inp.action;
        filter.actorId = inp.actorId;
        filter.after = inp.after;
        filter.before = inp.before;
        filter.limit = inp.limit > 0 ? inp.limit : 50;
        filter.offset = Math.max(inp.offset, 0);
        return audit.list(inp.applicationId, filter);
    }

    public AuditEvent getById(UUID id) {
        return audit.getById(id);
    }
}
