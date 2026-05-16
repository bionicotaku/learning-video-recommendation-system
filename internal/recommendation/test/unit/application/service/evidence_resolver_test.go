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
	rows       map[string][]model.SemanticSpan
	lastCtx    context.Context
	batchCalls int
}

type stubTranscriptSentenceReader struct {
	rows       map[string][]model.TranscriptSentence
	lastCtx    context.Context
	batchCalls int
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

func (s *stubSemanticSpanReader) ListByVideoUnitRefs(ctx context.Context, refs []apprepo.SemanticSpanRef) ([]model.SemanticSpan, error) {
	s.batchCalls++
	s.lastCtx = ctx
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	result := make([]model.SemanticSpan, 0, len(refs))
	for _, ref := range refs {
		for _, row := range s.rows[spanKey(ref.VideoID, ref.CoarseUnitID)] {
			if row.SentenceIndex == ref.Ref.SentenceIndex && row.SpanIndex == ref.Ref.SpanIndex {
				result = append(result, row)
			}
		}
	}
	return result, nil
}

func (s *stubTranscriptSentenceReader) ListByVideoAndIndexesBatch(ctx context.Context, refs []apprepo.TranscriptSentenceRef) ([]model.TranscriptSentence, error) {
	s.batchCalls++
	s.lastCtx = ctx
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	allowed := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		allowed[sentenceKey(ref.VideoID, ref.SentenceIndex)] = struct{}{}
	}

	result := make([]model.TranscriptSentence, 0, len(refs))
	for videoID, rows := range s.rows {
		for _, row := range rows {
			if _, ok := allowed[sentenceKey(videoID, row.SentenceIndex)]; ok {
				result = append(result, row)
			}
		}
	}
	return result, nil
}

func TestDefaultEvidenceResolverBatchesSpanAndSentenceReads(t *testing.T) {
	spanReader := &stubSemanticSpanReader{
		rows: map[string][]model.SemanticSpan{
			spanKey("video-1", 101): {
				{VideoID: "video-1", CoarseUnitID: int64Ptr(101), SentenceIndex: 1, SpanIndex: 1, StartMs: 1000, EndMs: 1400},
			},
			spanKey("video-2", 102): {
				{VideoID: "video-2", CoarseUnitID: int64Ptr(102), SentenceIndex: 2, SpanIndex: 1, StartMs: 2000, EndMs: 2400},
			},
		},
	}
	sentenceReader := &stubTranscriptSentenceReader{
		rows: map[string][]model.TranscriptSentence{
			"video-1": {{VideoID: "video-1", SentenceIndex: 1, StartMs: 900, EndMs: 1500}},
			"video-2": {{VideoID: "video-2", SentenceIndex: 2, StartMs: 1900, EndMs: 2500}},
		},
	}
	resolver := recommendationservice.NewDefaultEvidenceResolver(spanReader, sentenceReader)

	windows, err := resolver.Resolve(context.Background(), model.RecommendationContext{}, []model.VideoUnitCandidate{
		{
			VideoID:         "video-1",
			CoarseUnitID:    101,
			BestEvidenceRef: &model.EvidenceRef{SentenceIndex: 1, SpanIndex: 1},
		},
		{
			VideoID:         "video-2",
			CoarseUnitID:    102,
			BestEvidenceRef: &model.EvidenceRef{SentenceIndex: 2, SpanIndex: 1},
		},
	}, model.DemandBundle{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(windows) != 2 {
		t.Fatalf("resolved windows = %d, want 2", len(windows))
	}
	if spanReader.batchCalls != 1 {
		t.Fatalf("span batch calls = %d, want 1", spanReader.batchCalls)
	}
	if sentenceReader.batchCalls != 1 {
		t.Fatalf("sentence batch calls = %d, want 1", sentenceReader.batchCalls)
	}
}

func TestDefaultEvidenceResolverResolvesBestEvidenceSpanAndSentence(t *testing.T) {
	resolver := recommendationservice.NewDefaultEvidenceResolver(
		&stubSemanticSpanReader{
			rows: map[string][]model.SemanticSpan{
				spanKey("video-1", 101): {
					{VideoID: "video-1", CoarseUnitID: int64Ptr(101), SentenceIndex: 1, SpanIndex: 1, StartMs: 1000, EndMs: 1800},
					{VideoID: "video-1", CoarseUnitID: int64Ptr(101), SentenceIndex: 2, SpanIndex: 1, StartMs: 2100, EndMs: 2600},
				},
			},
		},
		&stubTranscriptSentenceReader{
			rows: map[string][]model.TranscriptSentence{
				"video-1": {
					{VideoID: "video-1", SentenceIndex: 1, StartMs: 900, EndMs: 1900},
					{VideoID: "video-1", SentenceIndex: 2, StartMs: 2000, EndMs: 2700},
				},
			},
		},
	)

	windows, err := resolver.Resolve(context.Background(), model.RecommendationContext{}, []model.VideoUnitCandidate{
		{
			VideoID:         "video-1",
			CoarseUnitID:    101,
			SentenceIndexes: []int32{1, 2},
			BestEvidenceRef: &model.EvidenceRef{SentenceIndex: 1, SpanIndex: 1},
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
	if len(window.ResolvedSentences) != 1 {
		t.Fatalf("expected one resolved sentence, got %#v", window.ResolvedSentences)
	}
	if window.WindowStartMs == nil || *window.WindowStartMs != 900 {
		t.Fatalf("expected sentence-backed window start, got %#v", window.WindowStartMs)
	}
}

func TestDefaultEvidenceResolverUsesExplicitBestEvidenceRef(t *testing.T) {
	resolver := recommendationservice.NewDefaultEvidenceResolver(
		&stubSemanticSpanReader{
			rows: map[string][]model.SemanticSpan{
				spanKey("video-2", 101): {
					{VideoID: "video-2", CoarseUnitID: int64Ptr(101), SentenceIndex: 1, SpanIndex: 1, StartMs: 1000, EndMs: 1400},
					{VideoID: "video-2", CoarseUnitID: int64Ptr(101), SentenceIndex: 1, SpanIndex: 2, StartMs: 1300, EndMs: 1600},
				},
			},
		},
		&stubTranscriptSentenceReader{},
	)

	windows, err := resolver.Resolve(context.Background(), model.RecommendationContext{}, []model.VideoUnitCandidate{
		{
			VideoID:         "video-2",
			CoarseUnitID:    101,
			BestEvidenceRef: &model.EvidenceRef{SentenceIndex: 1, SpanIndex: 2},
		},
	}, model.DemandBundle{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if windows[0].BestEvidenceRef == nil || windows[0].BestEvidenceRef.SpanIndex != 2 {
		t.Fatalf("expected explicit best evidence ref, got %#v", windows[0].BestEvidenceRef)
	}
}

func TestDefaultEvidenceResolverLeavesBestEvidenceEmptyWhenReferencedSpanIsMissing(t *testing.T) {
	resolver := recommendationservice.NewDefaultEvidenceResolver(
		&stubSemanticSpanReader{
			rows: map[string][]model.SemanticSpan{
				spanKey("video-3", 101): {
					{VideoID: "video-3", CoarseUnitID: int64Ptr(101), SentenceIndex: 3, SpanIndex: 1, StartMs: 3200, EndMs: 3600},
				},
			},
		},
		&stubTranscriptSentenceReader{
			rows: map[string][]model.TranscriptSentence{
				"video-3": {
					{VideoID: "video-3", SentenceIndex: 3, StartMs: 3000, EndMs: 3800},
				},
			},
		},
	)

	windows, err := resolver.Resolve(context.Background(), model.RecommendationContext{}, []model.VideoUnitCandidate{
		{
			VideoID:         "video-3",
			CoarseUnitID:    101,
			SentenceIndexes: []int32{3},
			BestEvidenceRef: &model.EvidenceRef{SentenceIndex: 9, SpanIndex: 9},
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
					{VideoID: "video-4", CoarseUnitID: int64Ptr(101), SentenceIndex: 1, SpanIndex: 1, StartMs: 1100, EndMs: 1700},
				},
			},
		},
		&stubTranscriptSentenceReader{},
	)

	windows, err := resolver.Resolve(context.Background(), model.RecommendationContext{}, []model.VideoUnitCandidate{
		{
			VideoID:         "video-4",
			CoarseUnitID:    101,
			SentenceIndexes: []int32{1},
			BestEvidenceRef: &model.EvidenceRef{SentenceIndex: 1, SpanIndex: 1},
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

func sentenceKey(videoID string, sentenceIndex int32) string {
	return fmt.Sprintf("%s#%d", videoID, sentenceIndex)
}

func TestDefaultEvidenceResolverPropagatesContextToReaders(t *testing.T) {
	spanReader := &stubSemanticSpanReader{
		rows: map[string][]model.SemanticSpan{
			spanKey("video-ctx", 101): {
				{VideoID: "video-ctx", CoarseUnitID: int64Ptr(101), SentenceIndex: 1, SpanIndex: 1, StartMs: 1000, EndMs: 1400},
			},
		},
	}
	sentenceReader := &stubTranscriptSentenceReader{
		rows: map[string][]model.TranscriptSentence{
			"video-ctx": {
				{VideoID: "video-ctx", SentenceIndex: 1, StartMs: 900, EndMs: 1500},
			},
		},
	}
	resolver := recommendationservice.NewDefaultEvidenceResolver(spanReader, sentenceReader)

	ctx := context.WithValue(context.Background(), "trace", "evidence-resolver")
	_, err := resolver.Resolve(ctx, model.RecommendationContext{}, []model.VideoUnitCandidate{
		{
			VideoID:         "video-ctx",
			CoarseUnitID:    101,
			SentenceIndexes: []int32{1},
			BestEvidenceRef: &model.EvidenceRef{SentenceIndex: 1, SpanIndex: 1},
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
