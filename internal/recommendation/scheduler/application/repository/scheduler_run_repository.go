package repository

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"
)

type SchedulerRunRepository interface {
	SaveRun(ctx context.Context, q sqlcgen.Querier, batch model.RecommendationBatch) error
	SaveRunItems(ctx context.Context, q sqlcgen.Querier, batch model.RecommendationBatch) error
}
