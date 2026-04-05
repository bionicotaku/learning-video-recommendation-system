package repository

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type SchedulerRunRepository interface {
	SaveRun(ctx context.Context, batch model.RecommendationBatch) error
	SaveRunItems(ctx context.Context, batch model.RecommendationBatch) error
}
