package resolver

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type EvidenceResolver interface {
	Resolve(ctx context.Context, context model.RecommendationContext, candidates []model.VideoUnitCandidate, demand model.DemandBundle) ([]model.ResolvedEvidenceWindow, error)
}
