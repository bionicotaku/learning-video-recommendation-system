package ranking

import "learning-video-recommendation-system/internal/recommendation/domain/model"

type VideoRanker interface {
	Rank(context model.RecommendationContext, candidates []model.VideoCandidate, demand model.DemandBundle) ([]model.VideoCandidate, error)
}
