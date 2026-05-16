package repository

import (
	"context"
	"encoding/json"

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

type semanticSpanRefJSON struct {
	VideoID       string `json:"video_id"`
	CoarseUnitID  int64  `json:"coarse_unit_id"`
	SentenceIndex int32  `json:"sentence_index"`
	SpanIndex     int32  `json:"span_index"`
}

type transcriptSentenceRefJSON struct {
	VideoID       string `json:"video_id"`
	SentenceIndex int32  `json:"sentence_index"`
}

func (r *SemanticSpanReader) ListByVideoUnitRefs(ctx context.Context, refs []apprepo.SemanticSpanRef) ([]model.SemanticSpan, error) {
	if len(refs) == 0 {
		return nil, nil
	}

	payload := make([]semanticSpanRefJSON, 0, len(refs))
	for _, ref := range refs {
		payload = append(payload, semanticSpanRefJSON{
			VideoID:       ref.VideoID,
			CoarseUnitID:  ref.CoarseUnitID,
			SentenceIndex: ref.Ref.SentenceIndex,
			SpanIndex:     ref.Ref.SpanIndex,
		})
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.ListSemanticSpansByRefs(ctx, rawPayload)
	if err != nil {
		return nil, err
	}

	result := make([]model.SemanticSpan, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapper.ToSemanticSpan(row))
	}
	return result, nil
}

func (r *TranscriptSentenceReader) ListByVideoAndIndexesBatch(ctx context.Context, refs []apprepo.TranscriptSentenceRef) ([]model.TranscriptSentence, error) {
	if len(refs) == 0 {
		return nil, nil
	}

	payload := make([]transcriptSentenceRefJSON, 0, len(refs))
	for _, ref := range refs {
		payload = append(payload, transcriptSentenceRefJSON{
			VideoID:       ref.VideoID,
			SentenceIndex: ref.SentenceIndex,
		})
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.ListTranscriptSentencesByRefs(ctx, rawPayload)
	if err != nil {
		return nil, err
	}

	result := make([]model.TranscriptSentence, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapper.ToTranscriptSentence(row))
	}
	return result, nil
}
