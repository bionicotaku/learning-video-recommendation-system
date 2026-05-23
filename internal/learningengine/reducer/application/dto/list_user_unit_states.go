package dto

import "learning-video-recommendation-system/internal/learningengine/reducer/domain/model"

type ListUserUnitStatesRequest struct {
	UserID     string
	OnlyTarget bool
}

type ListUserUnitStatesResponse struct {
	States []model.UserUnitState
}
