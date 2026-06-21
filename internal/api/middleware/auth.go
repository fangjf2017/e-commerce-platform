package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/ecommerce/feature-management/internal/domain"
)

type contextKey string

const (
	// ActorKey is the context key used to store the authenticated actor.
	ActorKey contextKey = "actor"
	// RequestIDKey is the context key used to store the request ID.
	RequestIDKey contextKey = "request_id"
)

// skippedPaths are paths that bypass authentication.
var skippedPaths = map[string]bool{
	"/healthz": true,
	"/readyz":  true,
	"/metrics": true,
}

// APIKeyAuth returns middleware that enforces Bearer token authentication.
// Health and metrics paths are exempt. On success the actor is stored in
// the request context under ActorKey.
func APIKeyAuth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skippedPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeUnauthorized(w, "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeUnauthorized(w, "invalid authorization header format")
				return
			}

			token := parts[1]
			if token != apiKey {
				writeUnauthorized(w, "invalid api key")
				return
			}

			actor := domain.AuditActor{
				ID:   "api-key",
				Type: "api_key",
			}
			ctx := context.WithValue(r.Context(), ActorKey, actor)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"` + message + `"}}`))
}
