package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ecommerce/feature-management/internal/observability"
)

// HTTPMetrics returns middleware that records Prometheus metrics for each request.
// It uses chi's route pattern for the path label so that parameterised paths are
// collapsed (e.g. /v1/flags/{flagKey} instead of /v1/flags/my-flag).
func HTTPMetrics(metrics *observability.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			wrapped := newResponseWriter(w)
			next.ServeHTTP(wrapped, r)

			duration := time.Since(start).Seconds()

			// Use the chi route pattern for a stable label value.
			pattern := ""
			if rctx := chi.RouteContext(r.Context()); rctx != nil {
				pattern = rctx.RoutePattern()
			}
			if pattern == "" {
				pattern = r.URL.Path
			}

			statusCode := fmt.Sprintf("%d", wrapped.status)

			metrics.HTTPRequestsTotal.WithLabelValues(r.Method, pattern, statusCode).Inc()
			metrics.HTTPRequestDuration.WithLabelValues(r.Method, pattern).Observe(duration)
		})
	}
}
