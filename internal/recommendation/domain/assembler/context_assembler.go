package assembler

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type ContextAssembler interface {
	Assemble(ctx context.Context, request model.RecommendationRequest) (model.RecommendationContext, error)
}
