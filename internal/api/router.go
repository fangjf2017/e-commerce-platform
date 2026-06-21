package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/ecommerce/feature-management/internal/api/handlers"
	"github.com/ecommerce/feature-management/internal/api/middleware"
	"github.com/ecommerce/feature-management/internal/observability"
	"github.com/ecommerce/feature-management/internal/service"
)

// NewRouter constructs the full chi router with all routes and middleware wired up.
func NewRouter(
	flagSvc *service.FlagService,
	evalSvc *service.EvaluationService,
	segmentSvc *service.SegmentService,
	auditSvc *service.AuditService,
	appSvc *service.ApplicationService,
	metrics *observability.Metrics,
	logger *zap.Logger,
	apiKey string,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware applied to every request.
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.RequestLogger(logger))
	r.Use(middleware.HTTPMetrics(metrics))
	r.Use(middleware.Tracing())
	r.Use(middleware.RateLimit(1000))
	r.Use(chiMiddleware.Compress(5))

	// Health and observability endpoints – no authentication required.
	healthH := handlers.NewHealthHandler()
	r.Get("/healthz", healthH.Liveness)
	r.Get("/readyz", healthH.Readiness)
	r.Handle("/metrics", promhttp.Handler())

	// All API routes require a valid API key.
	r.Group(func(r chi.Router) {
		r.Use(middleware.APIKeyAuth(apiKey))

		// Applications
		appH := handlers.NewApplicationHandler(appSvc)
		r.Route("/v1/applications", func(r chi.Router) {
			r.Post("/", appH.Create)
			r.Get("/", appH.List)
			r.Get("/{appID}", appH.Get)
			r.Patch("/{appID}", appH.Update)
			r.Delete("/{appID}", appH.Delete)

			// Environments
			envH := handlers.NewEnvironmentHandler(appSvc)
			r.Route("/{appID}/environments", func(r chi.Router) {
				r.Post("/", envH.Create)
				r.Get("/", envH.List)
				r.Get("/{envID}", envH.Get)
				r.Patch("/{envID}", envH.Update)
				r.Delete("/{envID}", envH.Delete)
			})

			// Flags
			flagH := handlers.NewFlagHandler(flagSvc)
			r.Route("/{appID}/environments/{envID}/flags", func(r chi.Router) {
				r.Post("/", flagH.Create)
				r.Get("/", flagH.List)
				r.Get("/{flagKey}", flagH.Get)
				r.Put("/{flagKey}", flagH.Update)
				r.Delete("/{flagKey}", flagH.Delete)
				r.Post("/{flagKey}/enable", flagH.Enable)
				r.Post("/{flagKey}/disable", flagH.Disable)
				r.Post("/{flagKey}/archive", flagH.Archive)

				// Rules
				ruleH := handlers.NewRuleHandler(flagSvc)
				r.Route("/{flagKey}/rules", func(r chi.Router) {
					r.Post("/", ruleH.Create)
					r.Get("/", ruleH.List)
					r.Put("/reorder", ruleH.Reorder)
					r.Put("/{ruleID}", ruleH.Update)
					r.Delete("/{ruleID}", ruleH.Delete)
				})
			})

			// Audit log
			auditH := handlers.NewAuditHandler(auditSvc)
			r.Route("/{appID}/audit", func(r chi.Router) {
				r.Get("/", auditH.List)
				r.Get("/{eventID}", auditH.Get)
			})

			// Segments
			segH := handlers.NewSegmentHandler(segmentSvc)
			r.Route("/{appID}/segments", func(r chi.Router) {
				r.Post("/", segH.Create)
				r.Get("/", segH.List)
				r.Get("/{segmentID}", segH.Get)
				r.Put("/{segmentID}", segH.Update)
				r.Delete("/{segmentID}", segH.Delete)
			})
		})

		// Evaluation endpoints
		evalH := handlers.NewEvaluationHandler(evalSvc)
		r.Route("/v1/evaluate", func(r chi.Router) {
			r.Post("/", evalH.Evaluate)
			r.Post("/batch", evalH.BatchEvaluate)
			r.Post("/dry-run", evalH.DryRun)
		})
	})

	return r
}
