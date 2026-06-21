package com.ecommerce.featuremanagement.service;

import com.ecommerce.featuremanagement.domain.BatchEvaluationRequest;
import com.ecommerce.featuremanagement.domain.EvaluationContext;
import com.ecommerce.featuremanagement.domain.EvaluationResult;
import com.ecommerce.featuremanagement.evaluator.Engine;
import org.springframework.stereotype.Service;

import java.util.List;

@Service
public class EvaluationService {

    private final Engine engine;

    public EvaluationService(Engine engine) {
        this.engine = engine;
    }

    public EvaluationResult evaluate(EvaluationContext ctx) {
        return engine.evaluate(ctx);
    }

    public List<EvaluationResult> batchEvaluate(BatchEvaluationRequest req) {
        return engine.batchEvaluate(req);
    }

    public EvaluationResult dryRun(EvaluationContext ctx) {
        return engine.evaluate(ctx);
    }
}
