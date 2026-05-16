package repository

import (
	"context"

	"learning-video-recommendation-system/internal/catalog/domain/model"
)

type EndQuizQuestionReader interface {
	HasVisibleVideoForEndQuiz(ctx context.Context, videoID string) (bool, error)
	ListVideoUnitQuizQuestionCandidates(ctx context.Context, videoID string, coarseUnitIDs []int64) ([]model.EndQuizQuestionCandidate, error)
	ListUnitQuizQuestionCandidates(ctx context.Context, coarseUnitIDs []int64) ([]model.EndQuizQuestionCandidate, error)
}
