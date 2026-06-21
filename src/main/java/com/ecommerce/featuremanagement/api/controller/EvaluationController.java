package com.ecommerce.featuremanagement.api.controller;

import com.ecommerce.featuremanagement.api.dto.ApiResponse;
import com.ecommerce.featuremanagement.domain.BatchEvaluationRequest;
import com.ecommerce.featuremanagement.domain.EvaluationContext;
import com.ecommerce.featuremanagement.domain.EvaluationResult;
import com.ecommerce.featuremanagement.service.EvaluationService;
import org.springframework.web.bind.annotation.*;

import java.time.Instant;
import java.util.List;

@RestController
@RequestMapping("/api/v1/evaluate")
public class EvaluationController {

    private final EvaluationService evaluation;

    public EvaluationController(EvaluationService evaluation) {
        this.evaluation = evaluation;
    }

    @PostMapping
    public ApiResponse<EvaluationResult> evaluate(@RequestBody EvaluationContext ctx) {
        if (ctx.timestamp == null) {
            ctx.timestamp = Instant.now();
        }
        return ApiResponse.of(evaluation.evaluate(ctx));
    }

    @PostMapping("/batch")
    public ApiResponse<List<EvaluationResult>> batch(@RequestBody BatchEvaluationRequest req) {
        return ApiResponse.of(evaluation.batchEvaluate(req));
    }

    @PostMapping("/dry-run")
    public ApiResponse<EvaluationResult> dryRun(@RequestBody EvaluationContext ctx) {
        if (ctx.timestamp == null) {
            ctx.timestamp = Instant.now();
        }
        return ApiResponse.of(evaluation.dryRun(ctx));
    }
}
