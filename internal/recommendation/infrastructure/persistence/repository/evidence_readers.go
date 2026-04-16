package repository

import (
	"context"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/mapper"
	recommendationsqlc "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/sqlcgen"
)

type SemanticSpanReader struct {
	queries *recommendationsqlc.Queries
}

type TranscriptSentenceReader struct {
	queries *recommendationsqlc.Queries
}

var _ apprepo.SemanticSpanReader = (*SemanticSpanReader)(nil)
var _ apprepo.TranscriptSentenceReader = (*TranscriptSentenceReader)(nil)

func NewSemanticSpanReader(db recommendationsqlc.DBTX) *SemanticSpanReader {
	return &SemanticSpanReader{queries: recommendationsqlc.New(db)}
}

func NewTranscriptSentenceReader(db recommendationsqlc.DBTX) *TranscriptSentenceReader {
	return &TranscriptSentenceReader{queries: recommendationsqlc.New(db)}
}

func (r *SemanticSpanReader) ListByVideoAndUnit(ctx context.Context, videoID string, coarseUnitID int64) ([]model.SemanticSpan, error) {
	pgVideoID, err := mapper.StringToUUID(videoID)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.ListSemanticSpansByVideoAndUnit(ctx, recommendationsqlc.ListSemanticSpansByVideoAndUnitParams{
		VideoID:      pgVideoID,
		CoarseUnitID: mapper.Int64PointerToPG(&coarseUnitID),
	})
	if err != nil {
		return nil, err
	}

	result := make([]model.SemanticSpan, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapper.ToSemanticSpan(row))
	}
	return result, nil
}

func (r *TranscriptSentenceReader) ListByVideoAndIndexes(ctx context.Context, videoID string, sentenceIndexes []int32) ([]model.TranscriptSentence, error) {
	pgVideoID, err := mapper.StringToUUID(videoID)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.ListTranscriptSentencesByVideoAndIndexes(ctx, recommendationsqlc.ListTranscriptSentencesByVideoAndIndexesParams{
		VideoID:         pgVideoID,
		SentenceIndexes: sentenceIndexes,
	})
	if err != nil {
		return nil, err
	}

	result := make([]model.TranscriptSentence, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapper.ToTranscriptSentence(row))
	}
	return result, nil
}
