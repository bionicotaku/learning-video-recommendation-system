package repository

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type SemanticSpanReader interface {
	ListByVideoAndUnit(ctx context.Context, videoID string, coarseUnitID int64) ([]model.SemanticSpan, error)
}

type TranscriptSentenceReader interface {
	ListByVideoAndIndexes(ctx context.Context, videoID string, sentenceIndexes []int32) ([]model.TranscriptSentence, error)
}
