package com.ecommerce.featuremanagement.api.controller;

import com.ecommerce.featuremanagement.api.dto.ApiMeta;
import com.ecommerce.featuremanagement.api.dto.ApiResponse;
import com.ecommerce.featuremanagement.domain.AuditActor;
import com.ecommerce.featuremanagement.domain.FeatureFlag;
import com.ecommerce.featuremanagement.domain.FlagStatus;
import com.ecommerce.featuremanagement.repository.PagedResult;
import com.ecommerce.featuremanagement.service.FlagService;
import com.ecommerce.featuremanagement.service.dto.CreateFlagInput;
import com.ecommerce.featuremanagement.service.dto.UpdateFlagInput;
import jakarta.servlet.http.HttpServletRequest;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.Map;
import java.util.UUID;

@RestController
@RequestMapping("/api/v1/applications/{appId}/environments/{envId}/flags")
public class FlagController {

    private final FlagService flags;

    public FlagController(FlagService flags) {
        this.flags = flags;
    }

    @PostMapping
    public ApiResponse<FeatureFlag> create(@PathVariable UUID appId, @PathVariable UUID envId,
                                            @RequestBody CreateFlagInput inp, HttpServletRequest req) {
        inp.applicationId = appId;
        inp.environmentId = envId;
        inp.actor = actor(req);
        return ApiResponse.of(flags.create(inp));
    }

    @GetMapping("/{key}")
    public ApiResponse<FeatureFlag> get(@PathVariable UUID appId, @PathVariable UUID envId, @PathVariable String key) {
        return ApiResponse.of(flags.get(appId, envId, key));
    }

    @GetMapping
    public ApiResponse<List<FeatureFlag>> list(@PathVariable UUID appId, @PathVariable UUID envId,
                                                @RequestParam(required = false) String status,
                                                @RequestParam(required = false) List<String> tags,
                                                @RequestParam(defaultValue = "20") int limit,
                                                @RequestParam(defaultValue = "0") int offset) {
        FlagStatus flagStatus = status == null ? null : FlagStatus.fromValue(status);
        PagedResult<FeatureFlag> result = flags.list(appId, envId, flagStatus, tags, limit, offset);
        return ApiResponse.of(result.items, new ApiMeta(result.total, limit, offset));
    }

    @PutMapping("/{key}")
    public ApiResponse<FeatureFlag> update(@PathVariable UUID appId, @PathVariable UUID envId, @PathVariable String key,
                                            @RequestBody UpdateFlagInput inp, HttpServletRequest req) {
        inp.actor = actor(req);
        return ApiResponse.of(flags.update(appId, envId, key, inp));
    }

    @DeleteMapping("/{key}")
    public void delete(@PathVariable UUID appId, @PathVariable UUID envId, @PathVariable String key, HttpServletRequest req) {
        flags.delete(appId, envId, key, actor(req));
    }

    @PatchMapping("/{key}/status")
    public ApiResponse<FeatureFlag> setStatus(@PathVariable UUID appId, @PathVariable UUID envId, @PathVariable String key,
                                               @RequestBody Map<String, String> body, HttpServletRequest req) {
        FlagStatus status = FlagStatus.fromValue(body.get("status"));
        return ApiResponse.of(flags.setStatus(appId, envId, key, status, actor(req)));
    }

    private AuditActor actor(HttpServletRequest req) {
        Object attr = req.getAttribute("actor");
        return attr instanceof AuditActor ? (AuditActor) attr : new AuditActor("unknown", null, "system");
    }
}
