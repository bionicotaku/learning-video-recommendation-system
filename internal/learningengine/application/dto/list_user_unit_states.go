package dto

import "learning-video-recommendation-system/internal/learningengine/domain/model"

type ListUserUnitStatesRequest struct {
	UserID string
}

type ListUserUnitStatesResponse struct {
	States []model.UserUnitState
}
