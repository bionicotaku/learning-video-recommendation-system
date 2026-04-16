package aggregator

import "learning-video-recommendation-system/internal/recommendation/domain/model"

type VideoEvidenceAggregator interface {
	Aggregate(context model.RecommendationContext, windows []model.ResolvedEvidenceWindow, demand model.DemandBundle) ([]model.VideoCandidate, error)
}
