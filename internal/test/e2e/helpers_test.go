//go:build e2e

package e2e

import (
	"context"
	"slices"
	"time"

	learningdto "learning-video-recommendation-system/internal/learningengine/application/dto"
	recommendationdto "learning-video-recommendation-system/internal/recommendation/application/dto"
	"learning-video-recommendation-system/internal/test/e2e/testutil"
)

func targetSpec(unitID int64, priority float64, ref string) learningdto.TargetUnitSpec {
	return learningdto.TargetUnitSpec{
		CoarseUnitID:      unitID,
		TargetSource:      "e2e",
		TargetSourceRefID: ref,
		TargetPriority:    priority,
	}
}

func ctx() context.Context {
	return context.Background()
}

func singleUnitVideo(videoID string, unitID int64, startMs, endMs int32, sentenceIndex int32, surface string, durationMs int32, mentionCount int32, coverageRatio float64, mappedSpanRatio float64) testutil.CatalogVideoFixture {
	return testutil.CatalogVideoFixture{
		VideoID:          videoID,
		DurationMs:       durationMs,
		Status:           "active",
		VisibilityStatus: "public",
		MappedSpanRatio:  mappedSpanRatio,
		TranscriptEntries: []testutil.TranscriptSentenceFixture{
			{SentenceIndex: sentenceIndex, Text: surface + " core", StartMs: startMs, EndMs: endMs},
			{SentenceIndex: sentenceIndex + 1, Text: surface + " support", StartMs: endMs, EndMs: endMs + 1_000},
		},
		SemanticSpans: []testutil.SemanticSpanFixture{
			{SentenceIndex: sentenceIndex, SpanIndex: 0, CoarseUnitID: &unitID, StartMs: startMs, EndMs: endMs, Text: surface},
		},
		UnitIndexes: []testutil.VideoUnitIndexFixture{
			{
				CoarseUnitID:       unitID,
				MentionCount:       mentionCount,
				SentenceCount:      2,
				FirstStartMs:       startMs,
				LastEndMs:          endMs + 1_000,
				CoverageMs:         endMs + 1_000 - startMs,
				CoverageRatio:      coverageRatio,
				SentenceIndexes:    []int32{sentenceIndex, sentenceIndex + 1},
				EvidenceSpanRefs:   []testutil.EvidenceRefFixture{{SentenceIndex: sentenceIndex, SpanIndex: 0}},
				SampleSurfaceForms: []string{surface},
			},
		},
	}
}

func strongSupplyVideo(videoID string, unitID int64, startMs, endMs int32, sentenceIndex int32, surface string, durationMs int32) testutil.CatalogVideoFixture {
	return singleUnitVideo(videoID, unitID, startMs, endMs, sentenceIndex, surface, durationMs, 3, 0.08, 0.82)
}

func weakSupplyVideo(videoID string, unitID int64, startMs, endMs int32, sentenceIndex int32, surface string, durationMs int32) testutil.CatalogVideoFixture {
	return singleUnitVideo(videoID, unitID, startMs, endMs, sentenceIndex, surface, durationMs, 1, 0.04, 0.40)
}

func hiddenVideo(fixture testutil.CatalogVideoFixture) testutil.CatalogVideoFixture {
	fixture.VisibilityStatus = "private"
	return fixture
}

func inactiveVideo(fixture testutil.CatalogVideoFixture) testutil.CatalogVideoFixture {
	fixture.Status = "inactive"
	return fixture
}

func futurePublishVideo(fixture testutil.CatalogVideoFixture, publishAt time.Time) testutil.CatalogVideoFixture {
	fixture.PublishAt = &publishAt
	return fixture
}

func bundleWeakSupportVideo(videoID string, entries []testutil.VideoUnitIndexFixture, spans []testutil.SemanticSpanFixture, sentences []testutil.TranscriptSentenceFixture, durationMs int32) testutil.CatalogVideoFixture {
	fixture := bundleVideo(videoID, entries, spans, sentences, durationMs, 0.55)
	fixture.MappedSpanRatio = 0.45
	return fixture
}

func bundleVideo(videoID string, entries []testutil.VideoUnitIndexFixture, spans []testutil.SemanticSpanFixture, sentences []testutil.TranscriptSentenceFixture, durationMs int32, mappedSpanRatio float64) testutil.CatalogVideoFixture {
	return testutil.CatalogVideoFixture{
		VideoID:           videoID,
		DurationMs:        durationMs,
		Status:            "active",
		VisibilityStatus:  "public",
		MappedSpanRatio:   mappedSpanRatio,
		TranscriptEntries: sentences,
		SemanticSpans:     spans,
		UnitIndexes:       entries,
	}
}

func mustRecordEvents(t interface {
	Helper()
	Fatalf(string, ...any)
}, suite *testutil.LearningSuite, userID string, events ...learningdto.LearningEventInput) {
	t.Helper()
	if _, err := suite.RecordEvents.Execute(ctx(), learningdto.RecordLearningEventsRequest{
		UserID: userID,
		Events: events,
	}); err != nil {
		t.Fatalf("RecordLearningEvents.Execute(): %v", err)
	}
}

func mustReplay(t interface {
	Helper()
	Fatalf(string, ...any)
}, suite *testutil.LearningSuite, userID string) {
	t.Helper()
	if _, err := suite.ReplayUserStates.Execute(ctx(), learningdto.ReplayUserStatesRequest{UserID: userID}); err != nil {
		t.Fatalf("ReplayUserStates.Execute(): %v", err)
	}
}

func recommendRequest(userID string, targetCount int) recommendationdto.GenerateVideoRecommendationsRequest {
	return recommendationdto.GenerateVideoRecommendationsRequest{
		UserID:               userID,
		TargetVideoCount:     targetCount,
		PreferredDurationSec: [2]int{45, 180},
		SessionHint:          "e2e",
		RequestContext:       []byte(`{"source":"e2e"}`),
	}
}

func mustRecommendN(t interface {
	Helper()
	Fatalf(string, ...any)
}, usecase interface {
	Execute(context.Context, recommendationdto.GenerateVideoRecommendationsRequest) (recommendationdto.GenerateVideoRecommendationsResponse, error)
}, userID string, targetCount int) recommendationdto.GenerateVideoRecommendationsResponse {
	t.Helper()
	response, err := usecase.Execute(ctx(), recommendRequest(userID, targetCount))
	if err != nil {
		t.Fatalf("GenerateVideoRecommendations.Execute(): %v", err)
	}
	return response
}

func assertSelectorMode(t interface {
	Helper()
	Fatalf(string, ...any)
}, response recommendationdto.GenerateVideoRecommendationsResponse, want string) {
	t.Helper()
	if response.SelectorMode != want {
		t.Fatalf("selector_mode = %q, want %q", response.SelectorMode, want)
	}
}

func assertContainsVideo(t interface {
	Helper()
	Fatalf(string, ...any)
}, videos []recommendationdto.RecommendationVideo, videoID string) {
	t.Helper()
	if videoIndex(videos, videoID) == -1 {
		t.Fatalf("expected video %s in result set, got %v", videoID, videoIDs(videos))
	}
}

func assertNotContainsVideo(t interface {
	Helper()
	Fatalf(string, ...any)
}, videos []recommendationdto.RecommendationVideo, videoID string) {
	t.Helper()
	if videoIndex(videos, videoID) != -1 {
		t.Fatalf("expected video %s to be excluded, got %v", videoID, videoIDs(videos))
	}
}

func assertCoveredUnits(t interface {
	Helper()
	Fatalf(string, ...any)
}, got []int64, want ...int64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("covered units = %v, want %v", got, want)
	}
	for _, unitID := range want {
		if !containsUnit(got, unitID) {
			t.Fatalf("covered units = %v, want %v", got, want)
		}
	}
}

func assertAnyVideoCoversUnit(t interface {
	Helper()
	Fatalf(string, ...any)
}, videos []recommendationdto.RecommendationVideo, unitID int64, selector func(testutil.RecommendationVideoView) []int64) {
	t.Helper()
	for _, video := range videos {
		if containsUnit(selector(video), unitID) {
			return
		}
	}
	t.Fatalf("expected some video to cover unit %d, got %+v", unitID, videos)
}

func assertContiguousRanks(t interface {
	Helper()
	Fatalf(string, ...any)
}, items []testutil.RecommendationItemSummary) {
	t.Helper()
	for index, item := range items {
		want := index + 1
		if item.Rank != want {
			t.Fatalf("item ranks are not contiguous: %+v", items)
		}
	}
}

func countPrimaryLane(items []testutil.RecommendationItemSummary, lane string) int {
	count := 0
	for _, item := range items {
		if item.PrimaryLane == lane {
			count++
		}
	}
	return count
}

func countDominantUnit(items []testutil.RecommendationItemSummary, unitID int64) int {
	count := 0
	for _, item := range items {
		if item.DominantUnitID != nil && *item.DominantUnitID == unitID {
			count++
		}
	}
	return count
}

func countCoreDominant(items []testutil.RecommendationItemSummary) int {
	count := 0
	for _, item := range items {
		if item.DominantBucket == "hard_review" || item.DominantBucket == "new_now" {
			count++
		}
	}
	return count
}

func countFutureLike(items []testutil.RecommendationItemSummary) int {
	count := 0
	for _, item := range items {
		if item.DominantBucket == "soft_review" || item.DominantBucket == "near_future" {
			count++
		}
	}
	return count
}

func videoIDs(videos []recommendationdto.RecommendationVideo) []string {
	result := make([]string, 0, len(videos))
	for _, video := range videos {
		result = append(result, video.VideoID)
	}
	return result
}

func videoIndex(videos []recommendationdto.RecommendationVideo, videoID string) int {
	for i, video := range videos {
		if video.VideoID == videoID {
			return i
		}
	}
	return -1
}

func containsUnit(units []int64, unitID int64) bool {
	return slices.Contains(units, unitID)
}

func containsReason(reasons []string, reason string) bool {
	return slices.Contains(reasons, reason)
}

func mustTimeAdd(base time.Time, delta time.Duration) time.Time {
	return base.UTC().Add(delta)
}
