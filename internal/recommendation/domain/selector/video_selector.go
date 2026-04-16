package selector

import "learning-video-recommendation-system/internal/recommendation/domain/model"

type VideoSelector interface {
	Select(context model.RecommendationContext, ranked []model.VideoCandidate, demand model.DemandBundle) ([]model.VideoCandidate, error)
}
