package com.ecommerce.featuremanagement.api.controller;

import com.ecommerce.featuremanagement.api.dto.ApiMeta;
import com.ecommerce.featuremanagement.api.dto.ApiResponse;
import com.ecommerce.featuremanagement.domain.AuditActor;
import com.ecommerce.featuremanagement.domain.Application;
import com.ecommerce.featuremanagement.domain.Environment;
import com.ecommerce.featuremanagement.repository.PagedResult;
import com.ecommerce.featuremanagement.service.ApplicationService;
import com.ecommerce.featuremanagement.service.dto.*;
import jakarta.servlet.http.HttpServletRequest;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.UUID;

@RestController
@RequestMapping("/api/v1")
public class ApplicationController {

    private final ApplicationService applications;

    public ApplicationController(ApplicationService applications) {
        this.applications = applications;
    }

    @PostMapping("/applications")
    public ApiResponse<Application> create(@RequestBody CreateApplicationInput inp, HttpServletRequest req) {
        inp.actor = actor(req);
        return ApiResponse.of(applications.create(inp));
    }

    @GetMapping("/applications/{id}")
    public ApiResponse<Application> get(@PathVariable UUID id) {
        return ApiResponse.of(applications.get(id));
    }

    @GetMapping("/applications")
    public ApiResponse<List<Application>> list(@RequestParam(defaultValue = "20") int limit,
                                                 @RequestParam(defaultValue = "0") int offset) {
        PagedResult<Application> result = applications.list(limit, offset);
        return ApiResponse.of(result.items, new ApiMeta(result.total, limit, offset));
    }

    @PutMapping("/applications/{id}")
    public ApiResponse<Application> update(@PathVariable UUID id, @RequestBody UpdateApplicationInput inp, HttpServletRequest req) {
        inp.actor = actor(req);
        return ApiResponse.of(applications.update(id, inp));
    }

    @DeleteMapping("/applications/{id}")
    public void delete(@PathVariable UUID id) {
        applications.delete(id);
    }

    @PostMapping("/applications/{appId}/environments")
    public ApiResponse<Environment> createEnvironment(@PathVariable UUID appId, @RequestBody CreateEnvironmentInput inp, HttpServletRequest req) {
        inp.applicationId = appId;
        inp.actor = actor(req);
        return ApiResponse.of(applications.createEnvironment(inp));
    }

    @GetMapping("/applications/{appId}/environments/{envId}")
    public ApiResponse<Environment> getEnvironment(@PathVariable UUID appId, @PathVariable UUID envId) {
        return ApiResponse.of(applications.getEnvironment(envId));
    }

    @GetMapping("/applications/{appId}/environments")
    public ApiResponse<List<Environment>> listEnvironments(@PathVariable UUID appId,
                                                             @RequestParam(defaultValue = "20") int limit,
                                                             @RequestParam(defaultValue = "0") int offset) {
        PagedResult<Environment> result = applications.listEnvironments(appId, limit, offset);
        return ApiResponse.of(result.items, new ApiMeta(result.total, limit, offset));
    }

    @PutMapping("/applications/{appId}/environments/{envId}")
    public ApiResponse<Environment> updateEnvironment(@PathVariable UUID appId, @PathVariable UUID envId,
                                                        @RequestBody UpdateEnvironmentInput inp, HttpServletRequest req) {
        inp.actor = actor(req);
        return ApiResponse.of(applications.updateEnvironment(envId, inp));
    }

    @DeleteMapping("/applications/{appId}/environments/{envId}")
    public void deleteEnvironment(@PathVariable UUID appId, @PathVariable UUID envId) {
        applications.deleteEnvironment(envId);
    }

    private AuditActor actor(HttpServletRequest req) {
        Object attr = req.getAttribute("actor");
        return attr instanceof AuditActor ? (AuditActor) attr : new AuditActor("unknown", null, "system");
    }
}
