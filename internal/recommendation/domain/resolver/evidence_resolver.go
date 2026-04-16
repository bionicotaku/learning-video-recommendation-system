package resolver

import "learning-video-recommendation-system/internal/recommendation/domain/model"

type EvidenceResolver interface {
	Resolve(context model.RecommendationContext, candidates []model.VideoUnitCandidate, demand model.DemandBundle) ([]model.ResolvedEvidenceWindow, error)
}
