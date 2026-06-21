package com.ecommerce.featuremanagement.api.controller;

import com.ecommerce.featuremanagement.api.dto.ApiMeta;
import com.ecommerce.featuremanagement.api.dto.ApiResponse;
import com.ecommerce.featuremanagement.domain.AuditActor;
import com.ecommerce.featuremanagement.domain.Segment;
import com.ecommerce.featuremanagement.repository.PagedResult;
import com.ecommerce.featuremanagement.service.SegmentService;
import com.ecommerce.featuremanagement.service.dto.CreateSegmentInput;
import com.ecommerce.featuremanagement.service.dto.UpdateSegmentInput;
import jakarta.servlet.http.HttpServletRequest;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.UUID;

@RestController
@RequestMapping("/api/v1")
public class SegmentController {

    private final SegmentService segments;

    public SegmentController(SegmentService segments) {
        this.segments = segments;
    }

    @PostMapping("/applications/{appId}/segments")
    public ApiResponse<Segment> create(@PathVariable UUID appId, @RequestBody CreateSegmentInput inp, HttpServletRequest req) {
        inp.applicationId = appId;
        inp.actor = actor(req);
        return ApiResponse.of(segments.create(inp));
    }

    @GetMapping("/segments/{id}")
    public ApiResponse<Segment> get(@PathVariable UUID id) {
        return ApiResponse.of(segments.get(id));
    }

    @GetMapping("/applications/{appId}/segments")
    public ApiResponse<List<Segment>> list(@PathVariable UUID appId,
                                            @RequestParam(defaultValue = "20") int limit,
                                            @RequestParam(defaultValue = "0") int offset) {
        PagedResult<Segment> result = segments.list(appId, limit, offset);
        return ApiResponse.of(result.items, new ApiMeta(result.total, limit, offset));
    }

    @PutMapping("/segments/{id}")
    public ApiResponse<Segment> update(@PathVariable UUID id, @RequestBody UpdateSegmentInput inp, HttpServletRequest req) {
        inp.actor = actor(req);
        return ApiResponse.of(segments.update(id, inp));
    }

    @DeleteMapping("/segments/{id}")
    public void delete(@PathVariable UUID id, HttpServletRequest req) {
        segments.delete(id, actor(req));
    }

    private AuditActor actor(HttpServletRequest req) {
        Object attr = req.getAttribute("actor");
        return attr instanceof AuditActor ? (AuditActor) attr : new AuditActor("unknown", null, "system");
    }
}
