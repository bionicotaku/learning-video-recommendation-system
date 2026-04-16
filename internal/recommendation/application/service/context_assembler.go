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
	defaultMaxDurationSec   = 180
)

type DefaultContextAssembler struct {
	learningStates apprepo.LearningStateReader
	inventory      apprepo.UnitInventoryReader
	unitServing    apprepo.UnitServingStateRepository
	videoServing   apprepo.VideoServingStateRepository
	videoUserState apprepo.VideoUserStateReader
	now            func() time.Time
}

var _ domainassembler.ContextAssembler = (*DefaultContextAssembler)(nil)

func NewDefaultContextAssembler(
	learningStates apprepo.LearningStateReader,
	inventory apprepo.UnitInventoryReader,
	unitServing apprepo.UnitServingStateRepository,
	videoServing apprepo.VideoServingStateRepository,
	videoUserState apprepo.VideoUserStateReader,
) *DefaultContextAssembler {
	return &DefaultContextAssembler{
		learningStates: learningStates,
		inventory:      inventory,
		unitServing:    unitServing,
		videoServing:   videoServing,
		videoUserState: videoUserState,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (a *DefaultContextAssembler) Assemble(ctx context.Context, request model.RecommendationRequest) (model.RecommendationContext, error) {
	normalized := normalizeRequest(request)

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
		Request:            normalized,
		Now:                a.now(),
		ActiveUnitStates:   states,
		UnitInventory:      inventory,
		UnitServingStates:  unitServingStates,
		VideoServingStates: []model.UserVideoServingState{},
		VideoUserStates:    []model.VideoUserState{},
	}, nil
}

func normalizeRequest(request model.RecommendationRequest) model.RecommendationRequest {
	result := request
	if result.TargetVideoCount <= 0 {
		result.TargetVideoCount = defaultTargetVideoCount
	}

	if result.PreferredDurationSec[0] <= 0 {
		result.PreferredDurationSec[0] = defaultMinDurationSec
	}
	if result.PreferredDurationSec[1] <= 0 {
		result.PreferredDurationSec[1] = defaultMaxDurationSec
	}
	if result.PreferredDurationSec[1] < result.PreferredDurationSec[0] {
		result.PreferredDurationSec[0] = defaultMinDurationSec
		result.PreferredDurationSec[1] = defaultMaxDurationSec
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
