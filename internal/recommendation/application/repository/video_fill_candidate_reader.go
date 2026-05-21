package repository

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type VideoFillCandidateReader interface {
	ListMasteredTargetFillCandidates(ctx context.Context, userID string, excludedVideoIDs []string, limit int32) ([]model.VideoFillCandidate, error)
	ListPopularFillCandidates(ctx context.Context, userID string, excludedVideoIDs []string, limit int32) ([]model.VideoFillCandidate, error)
}
