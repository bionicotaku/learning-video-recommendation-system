package service

import (
	"context"

	recommendationsqlc "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/sqlcgen"
)

type queriesContextKey struct{}

func WithQueries(ctx context.Context, queries *recommendationsqlc.Queries) context.Context {
	return context.WithValue(ctx, queriesContextKey{}, queries)
}

func queriesFromContext(ctx context.Context) (*recommendationsqlc.Queries, bool) {
	queries, ok := ctx.Value(queriesContextKey{}).(*recommendationsqlc.Queries)
	return queries, ok
}
