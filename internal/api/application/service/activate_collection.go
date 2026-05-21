package service

import (
	"context"
	"errors"

	apirepo "learning-video-recommendation-system/internal/api/application/repository"
	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	learningrepo "learning-video-recommendation-system/internal/learningengine/reducer/application/repository"
	learningservice "learning-video-recommendation-system/internal/learningengine/reducer/application/service"
	usermodel "learning-video-recommendation-system/internal/user/domain/model"
)

type ActivateLearningCollectionService struct {
	txManager apirepo.ActivateCollectionTxManager
}

func NewActivateLearningCollectionService(txManager apirepo.ActivateCollectionTxManager) *ActivateLearningCollectionService {
	return &ActivateLearningCollectionService{txManager: txManager}
}

func (s *ActivateLearningCollectionService) Execute(ctx context.Context, request learningdto.ActivateUnitCollectionTargetRequest) (learningdto.ActivateUnitCollectionTargetResponse, error) {
	if request.UserID == "" {
		return learningdto.ActivateUnitCollectionTargetResponse{}, InvalidRequestError("user_id is required")
	}
	if request.CollectionSlug == "" {
		return learningdto.ActivateUnitCollectionTargetResponse{}, InvalidRequestError("collection_slug is required")
	}
	if s.txManager == nil {
		return learningdto.ActivateUnitCollectionTargetResponse{}, errors.New("activate collection tx manager is required")
	}

	var response learningdto.ActivateUnitCollectionTargetResponse
	err := s.txManager.WithinUserTx(ctx, request.UserID, func(ctx context.Context, repos apirepo.ActivateCollectionRepositories) error {
		activated, err := repos.TargetCommands().ActivateUnitCollectionTarget(ctx, request.UserID, request.CollectionSlug)
		if err != nil {
			if errors.Is(err, learningrepo.ErrUnitCollectionNotFound) {
				return learningservice.ErrUnitCollectionNotFound
			}
			return err
		}

		if _, found, err := repos.UserProfiles().GetProfile(ctx, request.UserID); err != nil {
			return err
		} else if !found {
			if _, err := repos.UserProfiles().RepairProfile(ctx, request.UserID); err != nil {
				return err
			}
		}
		if err := repos.UserProfiles().UpdateOnboardingStatus(ctx, request.UserID, usermodel.OnboardingStatusCollectionSelected); err != nil {
			return err
		}

		response = learningdto.ActivateUnitCollectionTargetResponse{
			CollectionID:   activated.CollectionID,
			CollectionSlug: activated.CollectionSlug,
			TargetCount:    activated.TargetCount,
		}
		return nil
	})
	if err != nil {
		return learningdto.ActivateUnitCollectionTargetResponse{}, err
	}

	return response, nil
}
