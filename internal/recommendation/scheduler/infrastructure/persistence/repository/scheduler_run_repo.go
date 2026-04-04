package repository

import (
	"context"

	apprepo "learning-video-recommendation-system/internal/recommendation/scheduler/application/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/mapper"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"
)

type schedulerRunRepository struct{}

func NewSchedulerRunRepository() apprepo.SchedulerRunRepository {
	return schedulerRunRepository{}
}

func (schedulerRunRepository) SaveRun(ctx context.Context, q sqlcgen.Querier, batch model.RecommendationBatch) error {
	params, err := mapper.SchedulerRunParamsFromBatch(batch)
	if err != nil {
		return err
	}

	return q.InsertSchedulerRun(ctx, params)
}

func (schedulerRunRepository) SaveRunItems(ctx context.Context, q sqlcgen.Querier, batch model.RecommendationBatch) error {
	params, err := mapper.SchedulerRunItemParamsFromBatch(batch)
	if err != nil {
		return err
	}

	for _, item := range params {
		if err := q.InsertSchedulerRunItem(ctx, item); err != nil {
			return err
		}
	}

	return nil
}
