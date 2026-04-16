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

type EnsureTargetUnitsUsecase struct {
	txManager TxManager
}

var _ appusecase.EnsureTargetUnitsUsecase = (*EnsureTargetUnitsUsecase)(nil)

func NewEnsureTargetUnitsUsecase(txManager TxManager) *EnsureTargetUnitsUsecase {
	return &EnsureTargetUnitsUsecase{txManager: txManager}
}

func (u *EnsureTargetUnitsUsecase) Execute(ctx context.Context, request dto.EnsureTargetUnitsRequest) (dto.EnsureTargetUnitsResponse, error) {
	if request.UserID == "" {
		return dto.EnsureTargetUnitsResponse{}, fmt.Errorf("user_id is required")
	}

	targets := make([]model.TargetUnitSpec, 0, len(request.Targets))
	for _, target := range request.Targets {
		targets = append(targets, model.TargetUnitSpec{
			CoarseUnitID:      target.CoarseUnitID,
			TargetSource:      target.TargetSource,
			TargetSourceRefID: target.TargetSourceRefID,
			TargetPriority:    target.TargetPriority,
		})
	}

	err := u.txManager.WithinUserTx(ctx, request.UserID, func(ctx context.Context, repos TransactionalRepositories) error {
		return repos.TargetCommands().EnsureTargetUnits(ctx, request.UserID, targets)
	})
	if err != nil {
		return dto.EnsureTargetUnitsResponse{}, err
	}

	return dto.EnsureTargetUnitsResponse{TargetCount: len(targets)}, nil
}

type SetTargetInactiveUsecase struct {
	txManager TxManager
}

var _ appusecase.SetTargetInactiveUsecase = (*SetTargetInactiveUsecase)(nil)

func NewSetTargetInactiveUsecase(txManager TxManager) *SetTargetInactiveUsecase {
	return &SetTargetInactiveUsecase{txManager: txManager}
}

func (u *SetTargetInactiveUsecase) Execute(ctx context.Context, request dto.SetTargetInactiveRequest) (dto.SetTargetInactiveResponse, error) {
	if request.UserID == "" {
		return dto.SetTargetInactiveResponse{}, fmt.Errorf("user_id is required")
	}
	if request.CoarseUnitID == 0 {
		return dto.SetTargetInactiveResponse{}, fmt.Errorf("coarse_unit_id is required")
	}

	err := u.txManager.WithinUserTx(ctx, request.UserID, func(ctx context.Context, repos TransactionalRepositories) error {
		return repos.TargetCommands().SetTargetInactive(ctx, request.UserID, request.CoarseUnitID)
	})
	if err != nil {
		return dto.SetTargetInactiveResponse{}, err
	}

	return dto.SetTargetInactiveResponse{}, nil
}

type SuspendTargetUnitUsecase struct {
	txManager TxManager
}

var _ appusecase.SuspendTargetUnitUsecase = (*SuspendTargetUnitUsecase)(nil)

func NewSuspendTargetUnitUsecase(txManager TxManager) *SuspendTargetUnitUsecase {
	return &SuspendTargetUnitUsecase{txManager: txManager}
}

func (u *SuspendTargetUnitUsecase) Execute(ctx context.Context, request dto.SuspendTargetUnitRequest) (dto.SuspendTargetUnitResponse, error) {
	if request.UserID == "" {
		return dto.SuspendTargetUnitResponse{}, fmt.Errorf("user_id is required")
	}
	if request.CoarseUnitID == 0 {
		return dto.SuspendTargetUnitResponse{}, fmt.Errorf("coarse_unit_id is required")
	}

	err := u.txManager.WithinUserTx(ctx, request.UserID, func(ctx context.Context, repos TransactionalRepositories) error {
		state, err := repos.UserUnitStates().GetByUserAndUnitForUpdate(ctx, request.UserID, request.CoarseUnitID)
		if err != nil {
			return err
		}
		if state == nil {
			return ErrUserUnitStateNotFound
		}

		state.Status = enum.StatusSuspended
		state.SuspendedReason = request.SuspendedReason

		_, err = repos.UserUnitStates().Upsert(ctx, state)
		return err
	})
	if err != nil {
		return dto.SuspendTargetUnitResponse{}, err
	}

	return dto.SuspendTargetUnitResponse{}, nil
}

type ResumeTargetUnitUsecase struct {
	txManager TxManager
}

var _ appusecase.ResumeTargetUnitUsecase = (*ResumeTargetUnitUsecase)(nil)

func NewResumeTargetUnitUsecase(txManager TxManager) *ResumeTargetUnitUsecase {
	return &ResumeTargetUnitUsecase{txManager: txManager}
}

func (u *ResumeTargetUnitUsecase) Execute(ctx context.Context, request dto.ResumeTargetUnitRequest) (dto.ResumeTargetUnitResponse, error) {
	if request.UserID == "" {
		return dto.ResumeTargetUnitResponse{}, fmt.Errorf("user_id is required")
	}
	if request.CoarseUnitID == 0 {
		return dto.ResumeTargetUnitResponse{}, fmt.Errorf("coarse_unit_id is required")
	}

	err := u.txManager.WithinUserTx(ctx, request.UserID, func(ctx context.Context, repos TransactionalRepositories) error {
		state, err := repos.UserUnitStates().GetByUserAndUnitForUpdate(ctx, request.UserID, request.CoarseUnitID)
		if err != nil {
			return err
		}
		if state == nil {
			return ErrUserUnitStateNotFound
		}
		if state.Status != enum.StatusSuspended && state.SuspendedReason == "" {
			return ErrUserUnitStateNotSuspended
		}

		state.SuspendedReason = ""
		state.Status = aggregate.RecomputeActiveStatus(*state)

		_, err = repos.UserUnitStates().Upsert(ctx, state)
		return err
	})
	if err != nil {
		return dto.ResumeTargetUnitResponse{}, err
	}

	return dto.ResumeTargetUnitResponse{}, nil
}
