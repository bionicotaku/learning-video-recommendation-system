package candidate

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type CandidateGenerator interface {
	Generate(ctx context.Context, context model.RecommendationContext, demand model.DemandBundle) ([]model.VideoUnitCandidate, error)
}
