package middleware

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Tracing returns middleware that wraps each request in an OpenTelemetry span.
// The span name follows the pattern "HTTP {METHOD} {routePattern}".
func Tracing() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Determine the route pattern for the span name; fall back to path.
			pattern := r.URL.Path
			if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
				pattern = rctx.RoutePattern()
			}
			spanName := fmt.Sprintf("HTTP %s %s", r.Method, pattern)

			handler := otelhttp.NewHandler(next, spanName)
			handler.ServeHTTP(w, r)
		})
	}
}
