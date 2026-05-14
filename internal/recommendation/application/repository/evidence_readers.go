package repository

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type SemanticSpanReader interface {
	GetByVideoUnitAndRef(ctx context.Context, videoID string, coarseUnitID int64, ref model.EvidenceRef) (*model.SemanticSpan, error)
}

type TranscriptSentenceReader interface {
	ListByVideoAndIndexes(ctx context.Context, videoID string, sentenceIndexes []int32) ([]model.TranscriptSentence, error)
}
