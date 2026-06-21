package service

import (
	"context"
	"time"

	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/ecommerce/feature-management/internal/evaluator"
)

// EvaluationService wraps the evaluator Engine for flag evaluation.
type EvaluationService struct {
	engine *evaluator.Engine
}

// NewEvaluationService creates a new EvaluationService.
func NewEvaluationService(engine *evaluator.Engine) *EvaluationService {
	return &EvaluationService{engine: engine}
}

// Evaluate resolves a single feature flag for the provided context.
// Accepts a value (not pointer) to match the handler call site.
func (s *EvaluationService) Evaluate(ctx context.Context, req domain.EvaluationContext) (*domain.EvaluationResult, error) {
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now().UTC()
	}
	return s.engine.Evaluate(ctx, &req)
}

// BatchEvaluate resolves multiple feature flags for the same entity context.
// When FlagKeys is empty, all active flags for the environment are evaluated.
// Accepts a value (not pointer) to match the handler call site.
func (s *EvaluationService) BatchEvaluate(ctx context.Context, req domain.BatchEvaluationRequest) ([]*domain.EvaluationResult, error) {
	return s.engine.BatchEvaluate(ctx, &req)
}

// DryRun evaluates a flag without warming the cache — used for testing rules.
// It sets the timestamp to now and delegates to the engine.
// Accepts a value (not pointer) to match the handler call site.
func (s *EvaluationService) DryRun(ctx context.Context, req domain.EvaluationContext) (*domain.EvaluationResult, error) {
	req.Timestamp = time.Now().UTC()
	return s.engine.Evaluate(ctx, &req)
}
