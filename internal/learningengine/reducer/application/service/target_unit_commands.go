package service

import (
	"context"
	"errors"
	"fmt"

	"learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	apprepo "learning-video-recommendation-system/internal/learningengine/reducer/application/repository"
	appusecase "learning-video-recommendation-system/internal/learningengine/reducer/application/usecase"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/aggregate"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
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

type ActivateUnitCollectionTargetUsecase struct {
	txManager TxManager
}

var _ appusecase.ActivateUnitCollectionTargetUsecase = (*ActivateUnitCollectionTargetUsecase)(nil)

func NewActivateUnitCollectionTargetUsecase(txManager TxManager) *ActivateUnitCollectionTargetUsecase {
	return &ActivateUnitCollectionTargetUsecase{txManager: txManager}
}

func (u *ActivateUnitCollectionTargetUsecase) Execute(ctx context.Context, request dto.ActivateUnitCollectionTargetRequest) (dto.ActivateUnitCollectionTargetResponse, error) {
	if request.UserID == "" {
		return dto.ActivateUnitCollectionTargetResponse{}, fmt.Errorf("user_id is required")
	}
	if request.CollectionSlug == "" {
		return dto.ActivateUnitCollectionTargetResponse{}, validationError("collection_slug is required")
	}

	var activated model.ActivatedUnitCollectionTarget
	err := u.txManager.WithinUserTx(ctx, request.UserID, func(ctx context.Context, repos TransactionalRepositories) error {
		result, err := repos.TargetCommands().ActivateUnitCollectionTarget(ctx, request.UserID, request.CollectionSlug)
		if err != nil {
			return err
		}
		activated = result
		return nil
	})
	if err != nil {
		if errors.Is(err, apprepo.ErrUnitCollectionNotFound) {
			return dto.ActivateUnitCollectionTargetResponse{}, ErrUnitCollectionNotFound
		}
		return dto.ActivateUnitCollectionTargetResponse{}, err
	}

	return dto.ActivateUnitCollectionTargetResponse{
		CollectionID:   activated.CollectionID,
		CollectionSlug: activated.CollectionSlug,
		TargetCount:    activated.TargetCount,
	}, nil
}

type GetActiveUnitCollectionUsecase struct {
	reader apprepo.ActiveUnitCollectionReader
}

var _ appusecase.GetActiveUnitCollectionUsecase = (*GetActiveUnitCollectionUsecase)(nil)

func NewGetActiveUnitCollectionUsecase(reader apprepo.ActiveUnitCollectionReader) *GetActiveUnitCollectionUsecase {
	return &GetActiveUnitCollectionUsecase{reader: reader}
}

func (u *GetActiveUnitCollectionUsecase) Execute(ctx context.Context, request dto.GetActiveUnitCollectionRequest) (dto.GetActiveUnitCollectionResponse, error) {
	if request.UserID == "" {
		return dto.GetActiveUnitCollectionResponse{}, fmt.Errorf("user_id is required")
	}
	if u.reader == nil {
		return dto.GetActiveUnitCollectionResponse{}, fmt.Errorf("active collection reader is required")
	}

	active, err := u.reader.GetActiveUnitCollection(ctx, request.UserID)
	if err != nil {
		return dto.GetActiveUnitCollectionResponse{}, err
	}
	if active == nil {
		return dto.GetActiveUnitCollectionResponse{}, nil
	}
	return dto.GetActiveUnitCollectionResponse{
		ActiveCollection: &dto.ActiveUnitCollection{
			CollectionID:   active.CollectionID,
			CollectionSlug: active.CollectionSlug,
		},
	}, nil
}

type GetActiveLearningTargetCoarseUnitIDsUsecase struct {
	reader apprepo.ActiveLearningTargetReader
}

var _ appusecase.GetActiveLearningTargetCoarseUnitIDsUsecase = (*GetActiveLearningTargetCoarseUnitIDsUsecase)(nil)

func NewGetActiveLearningTargetCoarseUnitIDsUsecase(reader apprepo.ActiveLearningTargetReader) *GetActiveLearningTargetCoarseUnitIDsUsecase {
	return &GetActiveLearningTargetCoarseUnitIDsUsecase{reader: reader}
}

func (u *GetActiveLearningTargetCoarseUnitIDsUsecase) Execute(ctx context.Context, request dto.GetActiveLearningTargetCoarseUnitIDsRequest) (dto.GetActiveLearningTargetCoarseUnitIDsResponse, error) {
	if request.UserID == "" {
		return dto.GetActiveLearningTargetCoarseUnitIDsResponse{}, fmt.Errorf("user_id is required")
	}
	if u.reader == nil {
		return dto.GetActiveLearningTargetCoarseUnitIDsResponse{}, fmt.Errorf("active learning target reader is required")
	}

	targets, err := u.reader.GetActiveLearningTargetCoarseUnitIDs(ctx, request.UserID)
	if err != nil {
		return dto.GetActiveLearningTargetCoarseUnitIDsResponse{}, err
	}
	coarseUnitIDs := targets.CoarseUnitIDs
	if coarseUnitIDs == nil {
		coarseUnitIDs = []int64{}
	}
	return dto.GetActiveLearningTargetCoarseUnitIDsResponse{
		ActiveCollection: targets.ActiveCollection,
		TargetCount:      len(coarseUnitIDs),
		CoarseUnitIDs:    coarseUnitIDs,
	}, nil
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
