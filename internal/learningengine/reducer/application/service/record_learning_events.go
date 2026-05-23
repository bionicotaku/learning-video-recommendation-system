package service

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	appusecase "learning-video-recommendation-system/internal/learningengine/reducer/application/usecase"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/aggregate"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/policy"
)

type RecordLearningEventsUsecase struct {
	txManager TxManager
}

var _ appusecase.RecordLearningEventsUsecase = (*RecordLearningEventsUsecase)(nil)

func NewRecordLearningEventsUsecase(txManager TxManager) *RecordLearningEventsUsecase {
	return &RecordLearningEventsUsecase{txManager: txManager}
}

func (u *RecordLearningEventsUsecase) Execute(ctx context.Context, request dto.RecordLearningEventsRequest) (dto.RecordLearningEventsResponse, error) {
	if request.UserID == "" {
		return dto.RecordLearningEventsResponse{}, fmt.Errorf("user_id is required")
	}
	if len(request.Events) == 0 {
		return dto.RecordLearningEventsResponse{}, fmt.Errorf("events are required")
	}

	events := make([]model.LearningEvent, 0, len(request.Events))
	for _, input := range request.Events {
		metadata := input.Metadata
		if len(metadata) == 0 {
			metadata = []byte("{}")
		}

		event := model.LearningEvent{
			UserID:                    request.UserID,
			CoarseUnitID:              input.CoarseUnitID,
			VideoID:                   input.VideoID,
			EventType:                 input.EventType,
			ReducerEffect:             input.ReducerEffect,
			SourceType:                input.SourceType,
			SourceRefID:               input.SourceRefID,
			IsCorrect:                 input.IsCorrect,
			ProgressQuality:           input.ProgressQuality,
			CountsTowardSuccessStreak: input.CountsTowardSuccessStreak,
			ConsumedWatchSessionIDs:   append([]string(nil), input.ConsumedWatchSessionIDs...),
			Metadata:                  metadata,
			OccurredAt:                input.OccurredAt.UTC(),
			ResetBoundaryAt:           input.ResetBoundaryAt,
		}
		if err := policy.ValidateEvent(event); err != nil {
			return dto.RecordLearningEventsResponse{}, err
		}
		events = append(events, event)
	}

	groupedEvents := groupAndSortEvents(events)
	orderedEvents := flattenGroupedEvents(groupedEvents)
	response := dto.RecordLearningEventsResponse{ReceivedCount: len(orderedEvents)}

	err := u.txManager.WithinUserTx(ctx, request.UserID, func(ctx context.Context, repos TransactionalRepositories) error {
		coarseUnitIDs := sortedCoarseUnitIDs(groupedEvents)
		currentStates, err := repos.UserUnitStates().ListByUserAndUnitIDsForUpdate(ctx, request.UserID, coarseUnitIDs)
		if err != nil {
			return err
		}
		eventsToAppend := filterEventsAfterResetBoundary(orderedEvents, currentStates)
		response.SkippedBeforeResetCount = len(orderedEvents) - len(eventsToAppend)
		if len(eventsToAppend) == 0 {
			return nil
		}

		appendResult, err := repos.UnitLearningEvents().Append(ctx, eventsToAppend)
		if err != nil {
			return err
		}
		response.RecordedCount = len(appendResult.InsertedEvents)
		response.DuplicateCount = appendResult.DuplicateCount

		if len(appendResult.InsertedEvents) == 0 {
			return nil
		}

		groupedInsertedEvents := groupEventsPreserveOrder(appendResult.InsertedEvents)
		coarseUnitIDs = sortedCoarseUnitIDs(groupedInsertedEvents)

		nextStates := make([]*model.UserUnitState, 0, len(groupedInsertedEvents))
		startedUnitCount := 0
		for _, coarseUnitID := range coarseUnitIDs {
			currentState := currentStates[coarseUnitID]
			initialProgress := float64(0)
			if currentState != nil {
				initialProgress = currentState.ProgressPercent
			}
			unitEvents := groupedInsertedEvents[coarseUnitID]
			for _, event := range unitEvents {
				nextState, err := aggregate.Reduce(currentState, event)
				if err != nil {
					if errors.Is(err, aggregate.ErrLateProgressEvent) {
						return ErrLateProgressEvent
					}
					return err
				}
				applyLearningEventProjection(nextState, event)
				currentState = nextState
			}

			if currentState != nil {
				if initialProgress <= 0 && currentState.ProgressPercent > 0 {
					startedUnitCount++
				}
				nextStates = append(nextStates, currentState)
			}
		}

		if _, err := repos.UserUnitStates().BatchUpsert(ctx, nextStates); err != nil {
			return err
		}

		if startedUnitCount > 0 {
			stats := repos.ActivityStats()
			if stats != nil {
				for i := 0; i < startedUnitCount; i++ {
					if err := stats.IncrementStartedUnit(ctx, request.UserID); err != nil {
						return err
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		return dto.RecordLearningEventsResponse{}, err
	}

	return response, nil
}

func filterEventsAfterResetBoundary(events []model.LearningEvent, currentStates map[int64]*model.UserUnitState) []model.LearningEvent {
	filtered := make([]model.LearningEvent, 0, len(events))
	for _, event := range events {
		if !policy.IsResetUnlearnedEffect(event.ReducerEffect) {
			if state := currentStates[event.CoarseUnitID]; state != nil && state.LatestResetBoundaryAt != nil && !event.OccurredAt.After(*state.LatestResetBoundaryAt) {
				continue
			}
		}
		filtered = append(filtered, event)
	}
	return filtered
}

func groupAndSortEvents(events []model.LearningEvent) map[int64][]model.LearningEvent {
	grouped := make(map[int64][]model.LearningEvent, len(events))
	for _, event := range events {
		grouped[event.CoarseUnitID] = append(grouped[event.CoarseUnitID], event)
	}

	for coarseUnitID := range grouped {
		sort.SliceStable(grouped[coarseUnitID], func(i, j int) bool {
			return grouped[coarseUnitID][i].OccurredAt.Before(grouped[coarseUnitID][j].OccurredAt)
		})
	}

	return grouped
}

func groupEventsPreserveOrder(events []model.LearningEvent) map[int64][]model.LearningEvent {
	grouped := make(map[int64][]model.LearningEvent, len(events))
	for _, event := range events {
		grouped[event.CoarseUnitID] = append(grouped[event.CoarseUnitID], event)
	}
	return grouped
}

func flattenGroupedEvents(grouped map[int64][]model.LearningEvent) []model.LearningEvent {
	coarseUnitIDs := sortedCoarseUnitIDs(grouped)
	orderedEvents := make([]model.LearningEvent, 0)
	for _, coarseUnitID := range coarseUnitIDs {
		orderedEvents = append(orderedEvents, grouped[coarseUnitID]...)
	}

	return orderedEvents
}

func sortedCoarseUnitIDs(grouped map[int64][]model.LearningEvent) []int64 {
	coarseUnitIDs := make([]int64, 0, len(grouped))
	for coarseUnitID := range grouped {
		coarseUnitIDs = append(coarseUnitIDs, coarseUnitID)
	}
	sort.Slice(coarseUnitIDs, func(i, j int) bool {
		return coarseUnitIDs[i] < coarseUnitIDs[j]
	})
	return coarseUnitIDs
}
