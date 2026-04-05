package queryctx

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/sqlcgen"
)

type contextKey struct{}

// WithQuerier stores a transaction-scoped querier in context.
func WithQuerier(ctx context.Context, querier sqlcgen.Querier) context.Context {
	return context.WithValue(ctx, contextKey{}, querier)
}

// FromContext returns a transaction-scoped querier when present.
func FromContext(ctx context.Context) (sqlcgen.Querier, bool) {
	querier, ok := ctx.Value(contextKey{}).(sqlcgen.Querier)
	return querier, ok
}
