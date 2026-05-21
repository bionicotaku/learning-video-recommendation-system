package service

import (
	"context"
	"errors"

	"learning-video-recommendation-system/internal/user/application/dto"
	"learning-video-recommendation-system/internal/user/application/repository"
	"learning-video-recommendation-system/internal/user/domain/model"
)

type UpdateOnboardingStatusUsecase struct {
	profiles repository.ProfileRepository
}

func NewUpdateOnboardingStatusUsecase(profiles repository.ProfileRepository) *UpdateOnboardingStatusUsecase {
	return &UpdateOnboardingStatusUsecase{profiles: profiles}
}

func (u *UpdateOnboardingStatusUsecase) Execute(ctx context.Context, request dto.UpdateOnboardingStatusRequest) (dto.UpdateOnboardingStatusResponse, error) {
	if request.UserID == "" {
		return dto.UpdateOnboardingStatusResponse{}, ValidationError("user_id is required")
	}
	if !validOnboardingStatus(request.Status) {
		return dto.UpdateOnboardingStatusResponse{}, ValidationError("onboarding_status is invalid")
	}
	if u.profiles == nil {
		return dto.UpdateOnboardingStatusResponse{}, errors.New("profile repository is required")
	}
	if _, found, err := u.profiles.GetProfile(ctx, request.UserID); err != nil {
		return dto.UpdateOnboardingStatusResponse{}, err
	} else if !found {
		if _, err := u.profiles.RepairProfile(ctx, request.UserID); err != nil {
			return dto.UpdateOnboardingStatusResponse{}, err
		}
	}
	if err := u.profiles.UpdateOnboardingStatus(ctx, request.UserID, request.Status); err != nil {
		return dto.UpdateOnboardingStatusResponse{}, err
	}
	return dto.UpdateOnboardingStatusResponse{}, nil
}

func validOnboardingStatus(value string) bool {
	switch value {
	case model.OnboardingStatusNew, model.OnboardingStatusCollectionSelected, model.OnboardingStatusCompleted:
		return true
	default:
		return false
	}
}
