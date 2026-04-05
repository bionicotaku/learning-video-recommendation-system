package repository

import (
	"context"
	"fmt"

	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/queryctx"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"
)

func resolveQuerier(ctx context.Context, fallback sqlcgen.Querier) (sqlcgen.Querier, error) {
	if querier, ok := queryctx.FromContext(ctx); ok {
		return querier, nil
	}
	if fallback == nil {
		return nil, fmt.Errorf("repository querier is not configured")
	}

	return fallback, nil
}
