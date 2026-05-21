package service_test

import (
	"context"
	"testing"

	recommendationservice "learning-video-recommendation-system/internal/recommendation/application/service"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

func TestDefaultEvidenceResolverUsesRecallRowEvidenceDirectly(t *testing.T) {
	resolver := recommendationservice.NewDefaultEvidenceResolver()

	windows, err := resolver.Resolve(context.Background(), model.RecommendationContext{}, []model.VideoUnitCandidate{
		{
			VideoID:             "video-1",
			CoarseUnitID:        101,
			BestEvidenceRef:     &model.EvidenceRef{SentenceIndex: 2, SpanIndex: 3},
			BestEvidenceStartMs: 2300,
			BestEvidenceEndMs:   2600,
		},
	}, model.DemandBundle{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(windows) != 1 {
		t.Fatalf("resolved windows = %d, want 1", len(windows))
	}

	window := windows[0]
	if window.BestEvidenceRef == nil || window.BestEvidenceRef.SentenceIndex != 2 || window.BestEvidenceRef.SpanIndex != 3 {
		t.Fatalf("unexpected best evidence ref: %#v", window.BestEvidenceRef)
	}
	if window.BestEvidenceStartMs == nil || *window.BestEvidenceStartMs != 2300 {
		t.Fatalf("best evidence start = %#v, want 2300", window.BestEvidenceStartMs)
	}
	if window.BestEvidenceEndMs == nil || *window.BestEvidenceEndMs != 2600 {
		t.Fatalf("best evidence end = %#v, want 2600", window.BestEvidenceEndMs)
	}
	if len(window.WindowSentenceIndexes) != 1 || window.WindowSentenceIndexes[0] != 2 {
		t.Fatalf("window sentence indexes = %#v, want [2]", window.WindowSentenceIndexes)
	}
	if len(window.ResolvedSpans) != 0 || len(window.ResolvedSentences) != 0 {
		t.Fatalf("resolver should not hydrate spans/sentences, got spans=%#v sentences=%#v", window.ResolvedSpans, window.ResolvedSentences)
	}
}

func TestDefaultEvidenceResolverReturnsCanceledWhenContextCanceled(t *testing.T) {
	resolver := recommendationservice.NewDefaultEvidenceResolver()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := resolver.Resolve(ctx, model.RecommendationContext{}, []model.VideoUnitCandidate{
		{VideoID: "video-cancel", CoarseUnitID: 101},
	}, model.DemandBundle{})
	if err == nil || err != context.Canceled {
		t.Fatalf("resolve error = %v, want context.Canceled", err)
	}
}
