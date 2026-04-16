package service

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type ServingStateManager interface {
	ApplySelection(ctx context.Context, runID string, userID string, videos []model.FinalRecommendationItem) error
}

type AuditWriter interface {
	Write(ctx context.Context, run model.RecommendationRun, items []model.RecommendationItem) error
}

type RecommendationResultWriter interface {
	Persist(ctx context.Context, run model.RecommendationRun, items []model.RecommendationItem, userID string, videos []model.FinalRecommendationItem) error
}
