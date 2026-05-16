package repository

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type RecommendationAuditRepository interface {
	InsertRun(ctx context.Context, run model.RecommendationRun) error
	InsertItems(ctx context.Context, items []model.RecommendationItem) error
}
