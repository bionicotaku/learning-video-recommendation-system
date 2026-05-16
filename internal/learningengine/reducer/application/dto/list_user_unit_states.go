package dto

import "learning-video-recommendation-system/internal/learningengine/reducer/domain/model"

type ListUserUnitStatesRequest struct {
	UserID           string
	OnlyTarget       bool
	ExcludeSuspended bool
}

type ListUserUnitStatesResponse struct {
	States []model.UserUnitState
}
