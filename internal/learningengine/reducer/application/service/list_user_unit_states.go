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
var _ appusecase.GetUserUnitStateUsecase = (*GetUserUnitStateUsecase)(nil)

type GetUserUnitStateUsecase struct {
	userUnitStates apprepo.UserUnitStateRepository
}

func NewListUserUnitStatesUsecase(userUnitStates apprepo.UserUnitStateRepository) *ListUserUnitStatesUsecase {
	return &ListUserUnitStatesUsecase{userUnitStates: userUnitStates}
}

func NewGetUserUnitStateUsecase(userUnitStates apprepo.UserUnitStateRepository) *GetUserUnitStateUsecase {
	return &GetUserUnitStateUsecase{userUnitStates: userUnitStates}
}

func (u *GetUserUnitStateUsecase) Execute(ctx context.Context, request dto.GetUserUnitStateRequest) (dto.GetUserUnitStateResponse, error) {
	if request.UserID == "" {
		return dto.GetUserUnitStateResponse{}, fmt.Errorf("user_id is required")
	}
	if request.CoarseUnitID <= 0 {
		return dto.GetUserUnitStateResponse{}, fmt.Errorf("coarse_unit_id is required")
	}

	state, err := u.userUnitStates.GetByUserAndUnit(ctx, request.UserID, request.CoarseUnitID)
	if err != nil {
		return dto.GetUserUnitStateResponse{}, err
	}
	if state == nil {
		return dto.GetUserUnitStateResponse{Found: false}, nil
	}
	return dto.GetUserUnitStateResponse{Found: true, State: state}, nil
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
