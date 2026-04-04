package repository

import (
	"context"
	"time"

	apprepo "learning-video-recommendation-system/internal/recommendation/scheduler/application/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/mapper"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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

func (r unitLearningEventRepository) FindForReplay(ctx context.Context, userID uuid.UUID, coarseUnitID *int64, from *time.Time) ([]model.LearningEvent, error) {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return nil, err
	}

	var coarseUnitParam pgtype.Int8
	if coarseUnitID != nil {
		coarseUnitParam = pgtype.Int8{Int64: *coarseUnitID, Valid: true}
	}

	rows, err := q.FindUnitLearningEventsForReplay(ctx, sqlcgen.FindUnitLearningEventsForReplayParams{
		UserID:         mapper.UUIDToPG(userID),
		CoarseUnitID:   coarseUnitParam,
		FromOccurredAt: mapper.OptionalTimeToPG(from),
	})
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
