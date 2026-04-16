package candidate

import "learning-video-recommendation-system/internal/recommendation/domain/model"

type CandidateGenerator interface {
	Generate(context model.RecommendationContext, demand model.DemandBundle) ([]model.VideoUnitCandidate, error)
}
