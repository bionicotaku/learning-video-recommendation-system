package repository

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type LearningStateReader interface {
	ListActiveByUser(ctx context.Context, userID string) ([]model.LearningStateSnapshot, error)
}
