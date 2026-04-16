package service

import (
	"context"
	"fmt"

	"learning-video-recommendation-system/internal/learningengine/application/dto"
	appusecase "learning-video-recommendation-system/internal/learningengine/application/usecase"
	"learning-video-recommendation-system/internal/learningengine/domain/aggregate"
	"learning-video-recommendation-system/internal/learningengine/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
)

type ReplayUserStatesUsecase struct {
	txManager TxManager
}

var _ appusecase.ReplayUserStatesUsecase = (*ReplayUserStatesUsecase)(nil)

func NewReplayUserStatesUsecase(txManager TxManager) *ReplayUserStatesUsecase {
	return &ReplayUserStatesUsecase{txManager: txManager}
}

func (u *ReplayUserStatesUsecase) Execute(ctx context.Context, request dto.ReplayUserStatesRequest) (dto.ReplayUserStatesResponse, error) {
	if request.UserID == "" {
		return dto.ReplayUserStatesResponse{}, fmt.Errorf("user_id is required")
	}

	response := dto.ReplayUserStatesResponse{}

	err := u.txManager.WithinUserTx(ctx, request.UserID, func(ctx context.Context, repos TransactionalRepositories) error {
		currentStates, err := repos.UserUnitStates().ListByUser(ctx, request.UserID, model.UserUnitStateFilter{})
		if err != nil {
			return err
		}

		controlSnapshots := buildControlSnapshots(currentStates)

		events, err := repos.UnitLearningEvents().ListByUserOrdered(ctx, request.UserID)
		if err != nil {
			return err
		}
		response.ProcessedEventCount = len(events)

		if err := repos.UserUnitStates().DeleteByUser(ctx, request.UserID); err != nil {
			return err
		}

		rebuiltStates, err := replayStates(events)
		if err != nil {
			return err
		}

		finalStates := mergeSnapshots(request.UserID, rebuiltStates, controlSnapshots)
		response.RebuiltUnitCount = len(finalStates)

		if _, err := repos.UserUnitStates().BatchUpsert(ctx, finalStates); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return dto.ReplayUserStatesResponse{}, err
	}

	return response, nil
}

type controlSnapshot struct {
	UserID            string
	CoarseUnitID      int64
	IsTarget          bool
	TargetSource      string
	TargetSourceRefID string
	TargetPriority    float64
	Status            string
	SuspendedReason   string
	CreatedAt         *model.UserUnitState
}

func buildControlSnapshots(states []model.UserUnitState) map[int64]controlSnapshot {
	snapshots := make(map[int64]controlSnapshot, len(states))
	for _, state := range states {
		stateCopy := state
		snapshots[state.CoarseUnitID] = controlSnapshot{
			UserID:            state.UserID,
			CoarseUnitID:      state.CoarseUnitID,
			IsTarget:          state.IsTarget,
			TargetSource:      state.TargetSource,
			TargetSourceRefID: state.TargetSourceRefID,
			TargetPriority:    state.TargetPriority,
			Status:            state.Status,
			SuspendedReason:   state.SuspendedReason,
			CreatedAt:         &stateCopy,
		}
	}
	return snapshots
}

func replayStates(events []model.LearningEvent) (map[int64]*model.UserUnitState, error) {
	rebuilt := make(map[int64]*model.UserUnitState)
	for _, event := range events {
		nextState, err := aggregate.Reduce(rebuilt[event.CoarseUnitID], event)
		if err != nil {
			return nil, err
		}
		rebuilt[event.CoarseUnitID] = nextState
	}
	return rebuilt, nil
}

func mergeSnapshots(userID string, rebuilt map[int64]*model.UserUnitState, snapshots map[int64]controlSnapshot) []*model.UserUnitState {
	finalStates := make([]*model.UserUnitState, 0, len(snapshots)+len(rebuilt))

	for coarseUnitID, state := range rebuilt {
		if snapshot, ok := snapshots[coarseUnitID]; ok {
			applyControlSnapshot(state, snapshot)
			delete(snapshots, coarseUnitID)
		}
		finalStates = append(finalStates, state)
	}

	for _, snapshot := range snapshots {
		state := defaultStateFromSnapshot(userID, snapshot)
		finalStates = append(finalStates, state)
	}

	return finalStates
}

func applyControlSnapshot(state *model.UserUnitState, snapshot controlSnapshot) {
	state.IsTarget = snapshot.IsTarget
	state.TargetSource = snapshot.TargetSource
	state.TargetSourceRefID = snapshot.TargetSourceRefID
	state.TargetPriority = snapshot.TargetPriority
	state.SuspendedReason = snapshot.SuspendedReason
	if snapshot.Status == enum.StatusSuspended || snapshot.SuspendedReason != "" {
		state.Status = enum.StatusSuspended
	}
	if snapshot.CreatedAt != nil {
		state.CreatedAt = snapshot.CreatedAt.CreatedAt
	}
}

func defaultStateFromSnapshot(userID string, snapshot controlSnapshot) *model.UserUnitState {
	state := &model.UserUnitState{
		UserID:            userID,
		CoarseUnitID:      snapshot.CoarseUnitID,
		IsTarget:          snapshot.IsTarget,
		TargetSource:      snapshot.TargetSource,
		TargetSourceRefID: snapshot.TargetSourceRefID,
		TargetPriority:    snapshot.TargetPriority,
		Status:            enum.StatusNew,
		EaseFactor:        2.5,
	}
	if snapshot.Status == enum.StatusSuspended || snapshot.SuspendedReason != "" {
		state.Status = enum.StatusSuspended
		state.SuspendedReason = snapshot.SuspendedReason
	}
	if snapshot.CreatedAt != nil {
		state.CreatedAt = snapshot.CreatedAt.CreatedAt
	}
	return state
}
