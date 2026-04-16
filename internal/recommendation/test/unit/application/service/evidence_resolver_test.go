package service_test

import (
	"context"
	"fmt"
	"testing"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	recommendationservice "learning-video-recommendation-system/internal/recommendation/application/service"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type stubSemanticSpanReader struct {
	rows    map[string][]model.SemanticSpan
	lastCtx context.Context
}

type stubTranscriptSentenceReader struct {
	rows    map[string][]model.TranscriptSentence
	lastCtx context.Context
}

var _ apprepo.SemanticSpanReader = (*stubSemanticSpanReader)(nil)
var _ apprepo.TranscriptSentenceReader = (*stubTranscriptSentenceReader)(nil)

func (s *stubSemanticSpanReader) ListByVideoAndUnit(ctx context.Context, videoID string, coarseUnitID int64) ([]model.SemanticSpan, error) {
	s.lastCtx = ctx
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]model.SemanticSpan(nil), s.rows[spanKey(videoID, coarseUnitID)]...), nil
}

func (s *stubTranscriptSentenceReader) ListByVideoAndIndexes(ctx context.Context, videoID string, sentenceIndexes []int32) ([]model.TranscriptSentence, error) {
	s.lastCtx = ctx
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	allowed := make(map[int32]struct{}, len(sentenceIndexes))
	for _, index := range sentenceIndexes {
		allowed[index] = struct{}{}
	}

	rows := s.rows[videoID]
	result := make([]model.TranscriptSentence, 0, len(rows))
	for _, row := range rows {
		if _, ok := allowed[row.SentenceIndex]; ok {
			result = append(result, row)
		}
	}
	return result, nil
}

func TestDefaultEvidenceResolverResolvesReferencedSpansAndSentences(t *testing.T) {
	resolver := recommendationservice.NewDefaultEvidenceResolver(
		&stubSemanticSpanReader{
			rows: map[string][]model.SemanticSpan{
				spanKey("video-1", 101): {
					{VideoID: "video-1", CoarseUnitID: int64Ptr(101), SentenceIndex: 1, SpanIndex: 1, StartMs: 1000, EndMs: 1800, Text: "earlier"},
					{VideoID: "video-1", CoarseUnitID: int64Ptr(101), SentenceIndex: 2, SpanIndex: 1, StartMs: 2100, EndMs: 2600, Text: "later"},
				},
			},
		},
		&stubTranscriptSentenceReader{
			rows: map[string][]model.TranscriptSentence{
				"video-1": {
					{VideoID: "video-1", SentenceIndex: 1, StartMs: 900, EndMs: 1900, Text: "Sentence 1"},
					{VideoID: "video-1", SentenceIndex: 2, StartMs: 2000, EndMs: 2700, Text: "Sentence 2"},
				},
			},
		},
	)

	windows, err := resolver.Resolve(context.Background(), model.RecommendationContext{}, []model.VideoUnitCandidate{
		{
			VideoID:          "video-1",
			CoarseUnitID:     101,
			SentenceIndexes:  []int32{1, 2},
			EvidenceSpanRefs: []byte(`[{"sentence_index":1,"span_index":1},{"sentence_index":2,"span_index":1}]`),
		},
	}, model.DemandBundle{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if len(windows) != 1 {
		t.Fatalf("expected one resolved window, got %#v", windows)
	}
	window := windows[0]
	if window.BestEvidenceRef == nil || window.BestEvidenceRef.SentenceIndex != 1 || window.BestEvidenceRef.SpanIndex != 1 {
		t.Fatalf("unexpected best evidence ref: %#v", window.BestEvidenceRef)
	}
	if len(window.ResolvedSentences) != 2 {
		t.Fatalf("expected two resolved sentences, got %#v", window.ResolvedSentences)
	}
	if window.WindowStartMs == nil || *window.WindowStartMs != 900 {
		t.Fatalf("expected sentence-backed window start, got %#v", window.WindowStartMs)
	}
}

func TestDefaultEvidenceResolverPrefersEarlierSpanWhenRefsShareSentence(t *testing.T) {
	resolver := recommendationservice.NewDefaultEvidenceResolver(
		&stubSemanticSpanReader{
			rows: map[string][]model.SemanticSpan{
				spanKey("video-2", 101): {
					{VideoID: "video-2", CoarseUnitID: int64Ptr(101), SentenceIndex: 1, SpanIndex: 1, StartMs: 1000, EndMs: 1400, Text: "earlier"},
					{VideoID: "video-2", CoarseUnitID: int64Ptr(101), SentenceIndex: 1, SpanIndex: 2, StartMs: 1300, EndMs: 1600, Text: "later"},
				},
			},
		},
		&stubTranscriptSentenceReader{},
	)

	windows, err := resolver.Resolve(context.Background(), model.RecommendationContext{}, []model.VideoUnitCandidate{
		{
			VideoID:          "video-2",
			CoarseUnitID:     101,
			EvidenceSpanRefs: []byte(`[{"sentence_index":1,"span_index":2},{"sentence_index":1,"span_index":1}]`),
		},
	}, model.DemandBundle{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if windows[0].BestEvidenceRef == nil || windows[0].BestEvidenceRef.SpanIndex != 1 {
		t.Fatalf("expected earlier span in same sentence, got %#v", windows[0].BestEvidenceRef)
	}
}

func TestDefaultEvidenceResolverLeavesBestEvidenceEmptyWhenReferencedSpanIsMissing(t *testing.T) {
	resolver := recommendationservice.NewDefaultEvidenceResolver(
		&stubSemanticSpanReader{
			rows: map[string][]model.SemanticSpan{
				spanKey("video-3", 101): {
					{VideoID: "video-3", CoarseUnitID: int64Ptr(101), SentenceIndex: 3, SpanIndex: 1, StartMs: 3200, EndMs: 3600, Text: "fallback"},
				},
			},
		},
		&stubTranscriptSentenceReader{
			rows: map[string][]model.TranscriptSentence{
				"video-3": {
					{VideoID: "video-3", SentenceIndex: 3, StartMs: 3000, EndMs: 3800, Text: "Sentence 3"},
				},
			},
		},
	)

	windows, err := resolver.Resolve(context.Background(), model.RecommendationContext{}, []model.VideoUnitCandidate{
		{
			VideoID:          "video-3",
			CoarseUnitID:     101,
			SentenceIndexes:  []int32{3},
			EvidenceSpanRefs: []byte(`[{"sentence_index":9,"span_index":9}]`),
		},
	}, model.DemandBundle{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if windows[0].BestEvidenceRef != nil {
		t.Fatalf("expected missing best evidence ref, got %#v", windows[0].BestEvidenceRef)
	}
	if windows[0].BestEvidenceStartMs != nil || windows[0].BestEvidenceEndMs != nil {
		t.Fatalf("expected missing best evidence bounds, got start=%#v end=%#v", windows[0].BestEvidenceStartMs, windows[0].BestEvidenceEndMs)
	}
}

func TestDefaultEvidenceResolverToleratesMissingSentences(t *testing.T) {
	resolver := recommendationservice.NewDefaultEvidenceResolver(
		&stubSemanticSpanReader{
			rows: map[string][]model.SemanticSpan{
				spanKey("video-4", 101): {
					{VideoID: "video-4", CoarseUnitID: int64Ptr(101), SentenceIndex: 1, SpanIndex: 1, StartMs: 1100, EndMs: 1700, Text: "only-span"},
				},
			},
		},
		&stubTranscriptSentenceReader{},
	)

	windows, err := resolver.Resolve(context.Background(), model.RecommendationContext{}, []model.VideoUnitCandidate{
		{
			VideoID:          "video-4",
			CoarseUnitID:     101,
			SentenceIndexes:  []int32{1},
			EvidenceSpanRefs: []byte(`[{"sentence_index":1,"span_index":1}]`),
		},
	}, model.DemandBundle{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if len(windows[0].ResolvedSentences) != 0 {
		t.Fatalf("expected no resolved sentences, got %#v", windows[0].ResolvedSentences)
	}
	if windows[0].WindowStartMs == nil || *windows[0].WindowStartMs != 1100 {
		t.Fatalf("expected span-backed window start, got %#v", windows[0].WindowStartMs)
	}
}

func spanKey(videoID string, coarseUnitID int64) string {
	return fmt.Sprintf("%s#%d", videoID, coarseUnitID)
}

func TestDefaultEvidenceResolverPropagatesContextToReaders(t *testing.T) {
	spanReader := &stubSemanticSpanReader{
		rows: map[string][]model.SemanticSpan{
			spanKey("video-ctx", 101): {
				{VideoID: "video-ctx", CoarseUnitID: int64Ptr(101), SentenceIndex: 1, SpanIndex: 1, StartMs: 1000, EndMs: 1400, Text: "ctx"},
			},
		},
	}
	sentenceReader := &stubTranscriptSentenceReader{
		rows: map[string][]model.TranscriptSentence{
			"video-ctx": {
				{VideoID: "video-ctx", SentenceIndex: 1, StartMs: 900, EndMs: 1500, Text: "Sentence 1"},
			},
		},
	}
	resolver := recommendationservice.NewDefaultEvidenceResolver(spanReader, sentenceReader)

	ctx := context.WithValue(context.Background(), "trace", "evidence-resolver")
	_, err := resolver.Resolve(ctx, model.RecommendationContext{}, []model.VideoUnitCandidate{
		{
			VideoID:          "video-ctx",
			CoarseUnitID:     101,
			SentenceIndexes:  []int32{1},
			EvidenceSpanRefs: []byte(`[{"sentence_index":1,"span_index":1}]`),
		},
	}, model.DemandBundle{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if got := spanReader.lastCtx.Value("trace"); got != "evidence-resolver" {
		t.Fatalf("span reader ctx value = %#v, want propagated context", got)
	}
	if got := sentenceReader.lastCtx.Value("trace"); got != "evidence-resolver" {
		t.Fatalf("sentence reader ctx value = %#v, want propagated context", got)
	}
}

func TestDefaultEvidenceResolverReturnsCanceledWhenContextCanceled(t *testing.T) {
	resolver := recommendationservice.NewDefaultEvidenceResolver(&stubSemanticSpanReader{}, &stubTranscriptSentenceReader{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := resolver.Resolve(ctx, model.RecommendationContext{}, []model.VideoUnitCandidate{
		{VideoID: "video-cancel", CoarseUnitID: 101},
	}, model.DemandBundle{})
	if err == nil || err != context.Canceled {
		t.Fatalf("resolve error = %v, want context.Canceled", err)
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}
