package usecase

import (
	"context"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/application/command"
	"learning-video-recommendation-system/internal/recommendation/scheduler/application/dto"
	apprepo "learning-video-recommendation-system/internal/recommendation/scheduler/application/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/aggregate"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/policy"
)

type RecordLearningEventsAndUpdateStateUseCase struct {
	txManager apprepo.TxManager
	stateRepo apprepo.UserUnitStateRepository
	eventRepo apprepo.UnitLearningEventRepository
	reducer   aggregate.UserUnitReducer
}

func NewRecordLearningEventsAndUpdateStateUseCase(
	txManager apprepo.TxManager,
	stateRepo apprepo.UserUnitStateRepository,
	eventRepo apprepo.UnitLearningEventRepository,
	reducer aggregate.UserUnitReducer,
) RecordLearningEventsAndUpdateStateUseCase {
	return RecordLearningEventsAndUpdateStateUseCase{
		txManager: txManager,
		stateRepo: stateRepo,
		eventRepo: eventRepo,
		reducer:   reducer,
	}
}

func (uc RecordLearningEventsAndUpdateStateUseCase) Execute(ctx context.Context, cmd command.RecordLearningEventsCommand) (dto.RecordLearningEventsResult, error) {
	updatedUnits := make([]int64, 0, len(cmd.Events))
	seenUnits := make(map[int64]struct{}, len(cmd.Events))
	schedulerPolicy := policy.DefaultSchedulerPolicy()

	err := uc.txManager.WithinTx(ctx, func(ctx context.Context) error {
		for _, input := range cmd.Events {
			currentState, err := uc.stateRepo.GetByUserAndUnit(ctx, cmd.UserID, input.CoarseUnitID)
			if err != nil {
				return err
			}

			now := time.Now()
			event := model.LearningEvent{
				UserID:         cmd.UserID,
				CoarseUnitID:   input.CoarseUnitID,
				VideoID:        input.VideoID,
				EventType:      input.EventType,
				SourceType:     input.SourceType,
				SourceRefID:    input.SourceRefID,
				IsCorrect:      input.IsCorrect,
				Quality:        input.Quality,
				ResponseTimeMs: input.ResponseTimeMs,
				Metadata:       input.Metadata,
				OccurredAt:     nonZeroTime(input.OccurredAt, now),
				CreatedAt:      now,
			}

			if err := uc.eventRepo.Append(ctx, []model.LearningEvent{event}); err != nil {
				return err
			}

			nextState, err := uc.reducer.Reduce(currentState, event, schedulerPolicy)
			if err != nil {
				return err
			}
			if err := uc.stateRepo.Upsert(ctx, nextState); err != nil {
				return err
			}

			if _, ok := seenUnits[input.CoarseUnitID]; ok {
				continue
			}
			seenUnits[input.CoarseUnitID] = struct{}{}
			updatedUnits = append(updatedUnits, input.CoarseUnitID)
		}

		return nil
	})
	if err != nil {
		return dto.RecordLearningEventsResult{}, err
	}

	return dto.RecordLearningEventsResult{
		AcceptedCount: len(cmd.Events),
		UpdatedUnits:  updatedUnits,
	}, nil
}

func nonZeroTime(value, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}

	return value
}
