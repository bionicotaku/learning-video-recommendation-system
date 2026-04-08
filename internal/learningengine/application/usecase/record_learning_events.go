// 作用：实现在线写链路的主用例，负责在同一事务内写事件真相层并更新状态投影层。
// 输入/输出：输入是 RecordLearningEventsCommand；输出是 RecordLearningEventsResult 或 error。
// 谁调用它：上层业务调用方、integration/usecase/record_learning_events_usecase_test.go、fixture/helpers.go。
// 它调用谁/传给谁：调用 TxManager、UserUnitStateRepository、UnitLearningEventRepository、UserUnitReducer；最后把结果 DTO 返回给调用方。
package usecase

import (
	"context"
	"time"

	"learning-video-recommendation-system/internal/learningengine/application/command"
	"learning-video-recommendation-system/internal/learningengine/application/dto"
	apprepo "learning-video-recommendation-system/internal/learningengine/application/repository"
	"learning-video-recommendation-system/internal/learningengine/domain/aggregate"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	"learning-video-recommendation-system/internal/learningengine/domain/policy"
)

type RecordLearningEventsUseCase struct {
	txManager apprepo.TxManager
	stateRepo apprepo.UserUnitStateRepository
	eventRepo apprepo.UnitLearningEventRepository
	reducer   aggregate.UserUnitReducer
}

func NewRecordLearningEventsUseCase(
	txManager apprepo.TxManager,
	stateRepo apprepo.UserUnitStateRepository,
	eventRepo apprepo.UnitLearningEventRepository,
	reducer aggregate.UserUnitReducer,
) RecordLearningEventsUseCase {
	return RecordLearningEventsUseCase{
		txManager: txManager,
		stateRepo: stateRepo,
		eventRepo: eventRepo,
		reducer:   reducer,
	}
}

func (uc RecordLearningEventsUseCase) Execute(ctx context.Context, cmd command.RecordLearningEventsCommand) (dto.RecordLearningEventsResult, error) {
	updatedUnits := make([]int64, 0, len(cmd.Events))
	seenUnits := make(map[int64]struct{}, len(cmd.Events))
	schedulerPolicy := policy.DefaultLearningPolicy()

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
