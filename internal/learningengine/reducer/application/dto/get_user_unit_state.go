package dto

import "learning-video-recommendation-system/internal/learningengine/reducer/domain/model"

type GetUserUnitStateRequest struct {
	UserID       string
	CoarseUnitID int64
}

type GetUserUnitStateResponse struct {
	Found bool
	State *model.UserUnitState
}
