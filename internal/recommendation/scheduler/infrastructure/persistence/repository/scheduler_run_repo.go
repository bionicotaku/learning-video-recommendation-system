package repository

import (
	"context"

	apprepo "learning-video-recommendation-system/internal/recommendation/scheduler/application/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/mapper"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"
)

type schedulerRunRepository struct {
	querier sqlcgen.Querier
}

func NewSchedulerRunRepository(querier sqlcgen.Querier) apprepo.SchedulerRunRepository {
	return schedulerRunRepository{querier: querier}
}

func (r schedulerRunRepository) SaveRun(ctx context.Context, batch model.RecommendationBatch) error {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return err
	}

	params, err := mapper.SchedulerRunParamsFromBatch(batch)
	if err != nil {
		return err
	}

	return q.UpsertSchedulerRun(ctx, params)
}

func (r schedulerRunRepository) SaveRunItems(ctx context.Context, batch model.RecommendationBatch) error {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return err
	}

	params, err := mapper.SchedulerRunItemParamsFromBatch(batch)
	if err != nil {
		return err
	}

	for _, item := range params {
		if err := q.UpsertSchedulerRunItem(ctx, item); err != nil {
			return err
		}
	}

	return nil
}
