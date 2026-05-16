package service

import (
	"context"
	"fmt"

	"learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	apprepo "learning-video-recommendation-system/internal/learningengine/reducer/application/repository"
	appusecase "learning-video-recommendation-system/internal/learningengine/reducer/application/usecase"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
)

type ListUserUnitStatesUsecase struct {
	userUnitStates apprepo.UserUnitStateRepository
}

var _ appusecase.ListUserUnitStatesUsecase = (*ListUserUnitStatesUsecase)(nil)

func NewListUserUnitStatesUsecase(userUnitStates apprepo.UserUnitStateRepository) *ListUserUnitStatesUsecase {
	return &ListUserUnitStatesUsecase{userUnitStates: userUnitStates}
}

func (u *ListUserUnitStatesUsecase) Execute(ctx context.Context, request dto.ListUserUnitStatesRequest) (dto.ListUserUnitStatesResponse, error) {
	if request.UserID == "" {
		return dto.ListUserUnitStatesResponse{}, fmt.Errorf("user_id is required")
	}

	states, err := u.userUnitStates.ListByUser(ctx, request.UserID, model.UserUnitStateFilter{
		OnlyTarget:       request.OnlyTarget,
		ExcludeSuspended: request.ExcludeSuspended,
	})
	if err != nil {
		return dto.ListUserUnitStatesResponse{}, err
	}

	return dto.ListUserUnitStatesResponse{States: states}, nil
}
