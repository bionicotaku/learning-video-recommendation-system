package service

import (
	"context"
	"time"

	domainassembler "learning-video-recommendation-system/internal/recommendation/domain/assembler"
	"learning-video-recommendation-system/internal/recommendation/domain/model"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
)

const (
	defaultTargetVideoCount = 8
	defaultMinDurationSec   = 45
	defaultMaxDurationSec   = 200
)

type DefaultContextAssembler struct {
	learningStates apprepo.LearningStateReader
	inventory      apprepo.UnitInventoryReader
	unitServing    apprepo.UnitServingStateRepository
	recallQueue    *RecallQueueService
	recommendable  apprepo.RecommendableVideoUnitReader
	now            func() time.Time
}

var _ domainassembler.ContextAssembler = (*DefaultContextAssembler)(nil)

func NewDefaultContextAssembler(
	learningStates apprepo.LearningStateReader,
	inventory apprepo.UnitInventoryReader,
	unitServing apprepo.UnitServingStateRepository,
	recallQueue *RecallQueueService,
	recommendable apprepo.RecommendableVideoUnitReader,
) *DefaultContextAssembler {
	return &DefaultContextAssembler{
		learningStates: learningStates,
		inventory:      inventory,
		unitServing:    unitServing,
		recallQueue:    recallQueue,
		recommendable:  recommendable,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (a *DefaultContextAssembler) Assemble(ctx context.Context, request model.RecommendationRequest) (model.RecommendationContext, error) {
	normalized := normalizeRequest(request)
	now := a.now()

	if a.recallQueue != nil && a.recommendable != nil {
		return a.assembleFromRecallQueue(ctx, normalized, now)
	}

	states, err := a.learningStates.ListActiveByUser(ctx, normalized.UserID)
	if err != nil {
		return model.RecommendationContext{}, err
	}

	unitIDs := uniqueUnitIDs(states)

	var inventory []model.UnitVideoInventory
	if len(unitIDs) > 0 {
		inventory, err = a.inventory.ListByUnitIDs(ctx, unitIDs)
		if err != nil {
			return model.RecommendationContext{}, err
		}
	}

	var unitServingStates []model.UserUnitServingState
	if len(unitIDs) > 0 {
		unitServingStates, err = a.unitServing.ListByUserAndUnitIDs(ctx, normalized.UserID, unitIDs)
		if err != nil {
			return model.RecommendationContext{}, err
		}
	}

	return model.RecommendationContext{
		Request:              normalized,
		PreferredDurationSec: [2]int{defaultMinDurationSec, defaultMaxDurationSec},
		Now:                  now,
		ActiveUnitStates:     states,
		UnitInventory:        inventory,
		UnitServingStates:    unitServingStates,
		VideoServingStates:   []model.UserVideoServingState{},
		VideoUserStates:      []model.VideoUserState{},
	}, nil
}

func (a *DefaultContextAssembler) assembleFromRecallQueue(ctx context.Context, request model.RecommendationRequest, now time.Time) (model.RecommendationContext, error) {
	selection, err := a.recallQueue.SelectScope(ctx, request.UserID, request.TargetVideoCount, now)
	if err != nil {
		return model.RecommendationContext{}, err
	}
	summary := selection.Summary

	unitIDs := recallFetchScopeUnitIDs(selection.RecallFetchScope)
	var rows []model.RecommendableVideoUnit
	if len(unitIDs) > 0 {
		rows, err = a.recommendable.ListByUnitIDs(ctx, unitIDs, summary.PerUnitRecallLimit)
		if err != nil {
			return model.RecommendationContext{}, err
		}
	}
	summary.ActualRecallRowCount = len(rows)

	return model.RecommendationContext{
		Request:                request,
		PreferredDurationSec:   [2]int{defaultMinDurationSec, defaultMaxDurationSec},
		Now:                    now,
		ActiveUnitStates:       learningStatesFromRecallScope(selection.PlannerScope),
		UnitInventory:          inventoryFromRecallScope(selection.PlannerScope, now),
		UnitServingStates:      servingStatesFromRecallScope(selection.PlannerScope),
		VideoServingStates:     []model.UserVideoServingState{},
		VideoUserStates:        []model.VideoUserState{},
		RecommendableVideoUnit: rows,
		RecallScope:            summary,
	}, nil
}

func normalizeRequest(request model.RecommendationRequest) model.RecommendationRequest {
	result := request
	if result.TargetVideoCount <= 0 {
		result.TargetVideoCount = defaultTargetVideoCount
	}

	return result
}

func uniqueUnitIDs(states []model.LearningStateSnapshot) []int64 {
	seen := make(map[int64]struct{}, len(states))
	result := make([]int64, 0, len(states))
	for _, state := range states {
		if _, exists := seen[state.CoarseUnitID]; exists {
			continue
		}
		seen[state.CoarseUnitID] = struct{}{}
		result = append(result, state.CoarseUnitID)
	}
	return result
}

func recallFetchScopeUnitIDs(scope []model.RecallQueueCandidate) []int64 {
	result := make([]int64, 0, len(scope))
	for _, candidate := range scope {
		if candidate.SupplyGrade == "none" {
			continue
		}
		result = append(result, candidate.CoarseUnitID)
	}
	return result
}

func learningStatesFromRecallScope(scope []model.RecallQueueCandidate) []model.LearningStateSnapshot {
	result := make([]model.LearningStateSnapshot, 0, len(scope))
	for _, candidate := range scope {
		result = append(result, model.LearningStateSnapshot{
			UserID:              candidate.UserID,
			CoarseUnitID:        candidate.CoarseUnitID,
			IsTarget:            true,
			TargetPriority:      candidate.TargetPriority,
			Status:              candidate.Status,
			MasteryScore:        candidate.MasteryScore,
			LastProgressQuality: candidate.LastProgressQuality,
			NextReviewAt:        candidate.NextReviewAt,
			UpdatedAt:           candidate.StateUpdatedAt,
		})
	}
	return result
}

func inventoryFromRecallScope(scope []model.RecallQueueCandidate, now time.Time) []model.UnitVideoInventory {
	result := make([]model.UnitVideoInventory, 0, len(scope))
	for _, candidate := range scope {
		result = append(result, model.UnitVideoInventory{
			CoarseUnitID: candidate.CoarseUnitID,
			SupplyGrade:  candidate.SupplyGrade,
			UpdatedAt:    now,
		})
	}
	return result
}

func servingStatesFromRecallScope(scope []model.RecallQueueCandidate) []model.UserUnitServingState {
	result := make([]model.UserUnitServingState, 0, len(scope))
	for _, candidate := range scope {
		result = append(result, model.UserUnitServingState{
			UserID:       candidate.UserID,
			CoarseUnitID: candidate.CoarseUnitID,
			LastServedAt: candidate.LastServedAt,
			ServedCount:  candidate.ServedCount,
		})
	}
	return result
}
