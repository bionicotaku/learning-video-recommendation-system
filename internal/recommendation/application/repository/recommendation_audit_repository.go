package repository

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type RecommendationAuditRepository interface {
	InsertRun(ctx context.Context, run model.RecommendationRun) error
	InsertItem(ctx context.Context, item model.RecommendationItem) error
	InsertItems(ctx context.Context, items []model.RecommendationItem) error
}
