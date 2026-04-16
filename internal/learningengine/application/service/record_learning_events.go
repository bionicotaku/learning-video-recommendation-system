package service

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"learning-video-recommendation-system/internal/learningengine/application/dto"
	appusecase "learning-video-recommendation-system/internal/learningengine/application/usecase"
	"learning-video-recommendation-system/internal/learningengine/domain/aggregate"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	"learning-video-recommendation-system/internal/learningengine/domain/policy"
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
			UserID:         request.UserID,
			CoarseUnitID:   input.CoarseUnitID,
			VideoID:        input.VideoID,
			EventType:      input.EventType,
			SourceType:     input.SourceType,
			SourceRefID:    input.SourceRefID,
			IsCorrect:      input.IsCorrect,
			Quality:        input.Quality,
			ResponseTimeMs: input.ResponseTimeMs,
			Metadata:       metadata,
			OccurredAt:     input.OccurredAt,
		}
		if err := policy.ValidateEvent(event); err != nil {
			return dto.RecordLearningEventsResponse{}, err
		}
		events = append(events, event)
	}

	groupedEvents := groupAndSortEvents(events)
	orderedEvents := flattenGroupedEvents(groupedEvents)

	err := u.txManager.WithinUserTx(ctx, request.UserID, func(ctx context.Context, repos TransactionalRepositories) error {
		if err := repos.UnitLearningEvents().Append(ctx, orderedEvents); err != nil {
			return err
		}

		nextStates := make([]*model.UserUnitState, 0, len(groupedEvents))
		for coarseUnitID, unitEvents := range groupedEvents {
			state, err := repos.UserUnitStates().GetByUserAndUnitForUpdate(ctx, request.UserID, coarseUnitID)
			if err != nil {
				return err
			}

			currentState := state
			for _, event := range unitEvents {
				nextState, err := aggregate.Reduce(currentState, event)
				if err != nil {
					if errors.Is(err, aggregate.ErrLateStrongEvent) {
						return ErrLateStrongEvent
					}
					return err
				}
				currentState = nextState
			}

			if currentState != nil {
				nextStates = append(nextStates, currentState)
			}
		}

		if _, err := repos.UserUnitStates().BatchUpsert(ctx, nextStates); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return dto.RecordLearningEventsResponse{}, err
	}

	return dto.RecordLearningEventsResponse{RecordedCount: len(orderedEvents)}, nil
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

func flattenGroupedEvents(grouped map[int64][]model.LearningEvent) []model.LearningEvent {
	coarseUnitIDs := make([]int64, 0, len(grouped))
	for coarseUnitID := range grouped {
		coarseUnitIDs = append(coarseUnitIDs, coarseUnitID)
	}
	sort.Slice(coarseUnitIDs, func(i, j int) bool {
		return coarseUnitIDs[i] < coarseUnitIDs[j]
	})

	orderedEvents := make([]model.LearningEvent, 0)
	for _, coarseUnitID := range coarseUnitIDs {
		orderedEvents = append(orderedEvents, grouped[coarseUnitID]...)
	}

	return orderedEvents
}
