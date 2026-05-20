package repository

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
)

type UserUnitProgressReader interface {
	ListUserUnitProgress(ctx context.Context, query dto.ListUserUnitProgressQuery) ([]dto.UnitProgressItem, error)
}
