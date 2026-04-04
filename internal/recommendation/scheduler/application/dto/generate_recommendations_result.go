package dto

import "learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"

// GenerateRecommendationsResult returns the scheduler batch for downstream recommendation stages.
type GenerateRecommendationsResult struct {
	Batch model.RecommendationBatch
}
