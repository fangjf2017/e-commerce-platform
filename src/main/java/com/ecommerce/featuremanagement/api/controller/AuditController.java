package com.ecommerce.featuremanagement.api.controller;

import com.ecommerce.featuremanagement.api.dto.ApiMeta;
import com.ecommerce.featuremanagement.api.dto.ApiResponse;
import com.ecommerce.featuremanagement.domain.AuditEvent;
import com.ecommerce.featuremanagement.repository.PagedResult;
import com.ecommerce.featuremanagement.service.AuditService;
import com.ecommerce.featuremanagement.service.dto.ListAuditInput;
import org.springframework.web.bind.annotation.*;

import java.time.Instant;
import java.util.List;
import java.util.UUID;

@RestController
@RequestMapping("/api/v1/applications/{appId}/audit-events")
public class AuditController {

    private final AuditService audit;

    public AuditController(AuditService audit) {
        this.audit = audit;
    }

    @GetMapping
    public ApiResponse<List<AuditEvent>> list(@PathVariable UUID appId,
                                               @RequestParam(required = false) String resourceType,
                                               @RequestParam(required = false) String action,
                                               @RequestParam(required = false) String actorId,
                                               @RequestParam(required = false) Instant after,
                                               @RequestParam(required = false) Instant before,
                                               @RequestParam(defaultValue = "50") int limit,
                                               @RequestParam(defaultValue = "0") int offset) {
        ListAuditInput inp = new ListAuditInput();
        inp.applicationId = appId;
        inp.resourceType = resourceType;
        inp.action = action;
        inp.actorId = actorId;
        inp.after = after;
        inp.before = before;
        inp.limit = limit;
        inp.offset = offset;
        PagedResult<AuditEvent> result = audit.list(inp);
        return ApiResponse.of(result.items, new ApiMeta(result.total, limit, offset));
    }

    @GetMapping("/{id}")
    public ApiResponse<AuditEvent> get(@PathVariable UUID appId, @PathVariable UUID id) {
        return ApiResponse.of(audit.getById(id));
    }
}
