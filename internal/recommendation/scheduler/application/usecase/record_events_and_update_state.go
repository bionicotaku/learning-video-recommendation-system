package usecase

import (
	"context"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/application/command"
	"learning-video-recommendation-system/internal/recommendation/scheduler/application/dto"
	apprepo "learning-video-recommendation-system/internal/recommendation/scheduler/application/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/policy"
	domainservice "learning-video-recommendation-system/internal/recommendation/scheduler/domain/service"
)

type RecordLearningEventsAndUpdateStateUseCase struct {
	txManager    apprepo.TxManager
	stateRepo    apprepo.UserUnitStateRepository
	eventRepo    apprepo.UnitLearningEventRepository
	stateUpdater domainservice.StateUpdater
}

func NewRecordLearningEventsAndUpdateStateUseCase(
	txManager apprepo.TxManager,
	stateRepo apprepo.UserUnitStateRepository,
	eventRepo apprepo.UnitLearningEventRepository,
	stateUpdater domainservice.StateUpdater,
) RecordLearningEventsAndUpdateStateUseCase {
	return RecordLearningEventsAndUpdateStateUseCase{
		txManager:    txManager,
		stateRepo:    stateRepo,
		eventRepo:    eventRepo,
		stateUpdater: stateUpdater,
	}
}

func (uc RecordLearningEventsAndUpdateStateUseCase) Execute(ctx context.Context, cmd command.RecordLearningEventsCommand) (dto.RecordLearningEventsResult, error) {
	updatedUnits := make([]int64, 0, len(cmd.Events))
	seenUnits := make(map[int64]struct{}, len(cmd.Events))

	err := uc.txManager.WithinTx(ctx, func(ctx context.Context) error {
		for _, input := range cmd.Events {
			history, err := uc.eventRepo.FindForReplay(ctx, cmd.UserID, &input.CoarseUnitID, nil)
			if err != nil {
				return err
			}

			currentState, err := uc.stateRepo.GetByUserAndUnit(ctx, cmd.UserID, input.CoarseUnitID)
			if err != nil {
				return err
			}

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
				OccurredAt:     nonZeroTime(input.OccurredAt, time.Now()),
				CreatedAt:      time.Now(),
			}

			nextState, _, err := uc.stateUpdater.Apply(currentState, event, domainservice.UpdateContext{
				SchedulerPolicy:   policy.DefaultSchedulerPolicy(),
				RecentQualities:   recentQualitiesFromEvents(history),
				RecentCorrectness: recentCorrectnessFromEvents(history),
				Now:               event.CreatedAt,
			})
			if err != nil {
				return err
			}

			if err := uc.eventRepo.Append(ctx, []model.LearningEvent{event}); err != nil {
				return err
			}
			if err := uc.stateRepo.Upsert(ctx, nextState); err != nil {
				return err
			}

			if _, ok := seenUnits[input.CoarseUnitID]; !ok {
				seenUnits[input.CoarseUnitID] = struct{}{}
				updatedUnits = append(updatedUnits, input.CoarseUnitID)
			}
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

func recentQualitiesFromEvents(events []model.LearningEvent) []int {
	qualities := make([]int, 0, len(events))
	for _, event := range events {
		if event.Quality != nil {
			qualities = append(qualities, *event.Quality)
		}
	}

	return qualities
}

func recentCorrectnessFromEvents(events []model.LearningEvent) []bool {
	values := make([]bool, 0, len(events))
	for _, event := range events {
		if event.IsCorrect != nil {
			values = append(values, *event.IsCorrect)
		}
	}

	return values
}

func nonZeroTime(value, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}

	return value
}
