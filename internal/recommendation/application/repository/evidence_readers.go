package repository

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type SemanticSpanRef struct {
	VideoID      string
	CoarseUnitID int64
	Ref          model.EvidenceRef
}

type TranscriptSentenceRef struct {
	VideoID       string
	SentenceIndex int32
}

type SemanticSpanReader interface {
	ListByVideoUnitRefs(ctx context.Context, refs []SemanticSpanRef) ([]model.SemanticSpan, error)
}

type TranscriptSentenceReader interface {
	ListByVideoAndIndexesBatch(ctx context.Context, refs []TranscriptSentenceRef) ([]model.TranscriptSentence, error)
}
