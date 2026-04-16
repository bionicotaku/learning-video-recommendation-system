package explain

import "learning-video-recommendation-system/internal/recommendation/domain/model"

type ExplanationBuilder interface {
	Build(context model.RecommendationContext, selected []model.VideoCandidate, demand model.DemandBundle) ([]model.FinalRecommendationItem, error)
}
