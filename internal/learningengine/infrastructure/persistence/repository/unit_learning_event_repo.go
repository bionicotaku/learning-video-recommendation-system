package repository

import (
	"context"

	apprepo "learning-video-recommendation-system/internal/learningengine/application/repository"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	"learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/mapper"
	"learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/sqlcgen"

	"github.com/google/uuid"
)

type unitLearningEventRepository struct {
	querier sqlcgen.Querier
}

func NewUnitLearningEventRepository(querier sqlcgen.Querier) apprepo.UnitLearningEventRepository {
	return unitLearningEventRepository{querier: querier}
}

func (r unitLearningEventRepository) Append(ctx context.Context, events []model.LearningEvent) error {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return err
	}

	for _, event := range events {
		params, err := mapper.LearningEventToInsertParams(event)
		if err != nil {
			return err
		}
		if err := q.InsertUnitLearningEvent(ctx, params); err != nil {
			return err
		}
	}

	return nil
}

func (r unitLearningEventRepository) ListByUserOrdered(ctx context.Context, userID uuid.UUID) ([]model.LearningEvent, error) {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return nil, err
	}

	rows, err := q.ListUnitLearningEventsByUserOrdered(ctx, mapper.UUIDToPG(userID))
	if err != nil {
		return nil, err
	}

	items := make([]model.LearningEvent, 0, len(rows))
	for _, row := range rows {
		item, err := mapper.LearningEventFromRow(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}
