package service

import (
	"context"
	"time"

	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/ecommerce/feature-management/internal/evaluator"
	"github.com/ecommerce/feature-management/internal/observability"
	"github.com/google/uuid"
)

// EvaluationService wraps the evaluator engine and records metrics.
type EvaluationService struct {
	engine  *evaluator.Engine
	metrics *observability.Metrics
}

// NewEvaluationService creates a new EvaluationService.
func NewEvaluationService(engine *evaluator.Engine, metrics *observability.Metrics) *EvaluationService {
	return &EvaluationService{engine: engine, metrics: metrics}
}

// Evaluate evaluates a single feature flag.
func (s *EvaluationService) Evaluate(ctx context.Context, evalCtx *domain.EvaluationContext) (*domain.EvaluationResult, error) {
	start := time.Now()

	result, err := s.engine.Evaluate(ctx, evalCtx)
	if err != nil {
		if s.metrics != nil {
			s.metrics.FlagEvaluationsTotal.WithLabelValues(
				evalCtx.FlagKey,
				evalCtx.EnvironmentID.String(),
				"error",
				"",
			).Inc()
		}
		return nil, err
	}

	duration := time.Since(start).Seconds()

	if s.metrics != nil {
		resultLabel := "false"
		if result.Enabled {
			resultLabel = "true"
		}
		s.metrics.FlagEvaluationsTotal.WithLabelValues(
			result.FlagKey,
			evalCtx.EnvironmentID.String(),
			resultLabel,
			result.CacheLayer,
		).Inc()
		s.metrics.EvaluationDuration.WithLabelValues(
			result.FlagKey,
			evalCtx.EnvironmentID.String(),
		).Observe(duration)
	}

	return result, nil
}

// BatchEvaluate evaluates multiple feature flags for a single entity.
func (s *EvaluationService) BatchEvaluate(ctx context.Context, req *domain.BatchEvaluationRequest) ([]*domain.EvaluationResult, error) {
	return s.engine.BatchEvaluate(ctx, req)
}

// EvaluateByKey is a convenience method that builds the EvaluationContext from parts.
func (s *EvaluationService) EvaluateByKey(
	ctx context.Context,
	appID, envID uuid.UUID,
	flagKey, entityID, entityType string,
	attributes map[string]interface{},
) (*domain.EvaluationResult, error) {
	evalCtx := &domain.EvaluationContext{
		FlagKey:       flagKey,
		ApplicationID: appID,
		EnvironmentID: envID,
		EntityID:      entityID,
		EntityType:    entityType,
		Attributes:    attributes,
		Timestamp:     time.Now().UTC(),
	}
	return s.Evaluate(ctx, evalCtx)
}
