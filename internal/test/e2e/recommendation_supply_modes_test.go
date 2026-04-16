//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	learningdto "learning-video-recommendation-system/internal/learningengine/application/dto"
	"learning-video-recommendation-system/internal/test/e2e/testutil"
)

func TestE2E_RecommendationNormalModeWithBundleCoverage(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()
	recommendation := h.RecommendationUsecase()

	userID := h.NewUserID()
	hardUnit := h.NewUnitID()
	newUnit := h.NewUnitID()
	softUnit := h.NewUnitID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, hardUnit, newUnit, softUnit)

	exactHard := h.NewVideoID()
	exactNew := h.NewVideoID()
	bundleID := h.NewVideoID()
	softVideo := h.NewVideoID()

	h.SeedCatalogVideo(t, singleUnitVideo(exactHard, hardUnit, 1_000, 2_200, 0, "hard-normal", 90_000, 4, 0.09, 0.84))
	h.SeedCatalogVideo(t, singleUnitVideo(exactNew, newUnit, 3_000, 4_100, 2, "new-normal", 90_000, 4, 0.09, 0.84))
	h.SeedCatalogVideo(t, singleUnitVideo(softVideo, softUnit, 7_000, 8_300, 6, "soft-normal", 90_000, 3, 0.07, 0.82))

	h.SeedCatalogVideo(t, bundleVideo(
		bundleID,
		[]testutil.VideoUnitIndexFixture{
			{
				CoarseUnitID:       hardUnit,
				MentionCount:       3,
				SentenceCount:      1,
				FirstStartMs:       1_200,
				LastEndMs:          2_300,
				CoverageMs:         1_100,
				CoverageRatio:      0.06,
				SentenceIndexes:    []int32{0},
				EvidenceSpanRefs:   []testutil.EvidenceRefFixture{{SentenceIndex: 0, SpanIndex: 0}},
				SampleSurfaceForms: []string{"hard bundle"},
			},
			{
				CoarseUnitID:       newUnit,
				MentionCount:       3,
				SentenceCount:      1,
				FirstStartMs:       4_200,
				LastEndMs:          5_300,
				CoverageMs:         1_100,
				CoverageRatio:      0.06,
				SentenceIndexes:    []int32{2},
				EvidenceSpanRefs:   []testutil.EvidenceRefFixture{{SentenceIndex: 2, SpanIndex: 0}},
				SampleSurfaceForms: []string{"new bundle"},
			},
		},
		[]testutil.SemanticSpanFixture{
			{SentenceIndex: 0, SpanIndex: 0, CoarseUnitID: &hardUnit, StartMs: 1_200, EndMs: 2_300, Text: "hard bundle"},
			{SentenceIndex: 2, SpanIndex: 0, CoarseUnitID: &newUnit, StartMs: 4_200, EndMs: 5_300, Text: "new bundle"},
		},
		[]testutil.TranscriptSentenceFixture{
			{SentenceIndex: 0, Text: "hard bundle sentence", StartMs: 1_200, EndMs: 2_300},
			{SentenceIndex: 1, Text: "bundle bridge", StartMs: 2_400, EndMs: 3_200},
			{SentenceIndex: 2, Text: "new bundle sentence", StartMs: 4_200, EndMs: 5_300},
		},
		120_000,
		0.88,
	))
	h.RefreshRecommendationViews(t)

	testutil.MustEnsureTarget(t, learning, userID,
		targetSpec(hardUnit, 0.95, "hard"),
		targetSpec(newUnit, 0.85, "new"),
		targetSpec(softUnit, 0.60, "soft"),
	)

	now := time.Now().UTC()
	q4 := int16(4)
	q2 := int16(2)
	if _, err := learning.RecordEvents.Execute(context.Background(), learningdto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []learningdto.LearningEventInput{
			{CoarseUnitID: hardUnit, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: now.Add(-48 * time.Hour)},
			{CoarseUnitID: hardUnit, EventType: "review", SourceType: "quiz_session", Quality: &q2, OccurredAt: now.Add(-24 * time.Hour)},
			{CoarseUnitID: softUnit, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: now.Add(-2 * time.Hour)},
		},
	}); err != nil {
		t.Fatalf("RecordLearningEvents.Execute(): %v", err)
	}

	response := testutil.MustRecommend(t, recommendation, userID, 3)
	if response.SelectorMode != "normal" {
		t.Fatalf("selector_mode = %q, want normal", response.SelectorMode)
	}
	if len(response.Videos) == 0 {
		t.Fatal("expected non-empty response")
	}
	if videoIndex(response.Videos, bundleID) == -1 {
		t.Fatalf("expected bundle video in result set, got %v", videoIDs(response.Videos))
	}
	bundle := response.Videos[videoIndex(response.Videos, bundleID)]
	if !containsUnit(bundle.CoveredHardReviewUnits, hardUnit) || !containsUnit(bundle.CoveredNewNowUnits, newUnit) {
		t.Fatalf("bundle coverage incomplete: %+v", bundle)
	}
}

func TestE2E_RecommendationLowSupplyModePreservesCoreCoverage(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()
	recommendation := h.RecommendationUsecase()

	userID := h.NewUserID()
	hardUnit := h.NewUnitID()
	softUnitA := h.NewUnitID()
	softUnitB := h.NewUnitID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, hardUnit, softUnitA, softUnitB)

	weakHardVideo := h.NewVideoID()
	bundleSupport := h.NewVideoID()
	softVideo := h.NewVideoID()

	h.SeedCatalogVideo(t, singleUnitVideo(weakHardVideo, hardUnit, 1_000, 2_100, 0, "hard-weak", 90_000, 1, 0.05, 0.60))
	h.SeedCatalogVideo(t, singleUnitVideo(softVideo, softUnitA, 7_000, 8_100, 4, "soft-a", 90_000, 3, 0.07, 0.82))
	h.SeedCatalogVideo(t, bundleVideo(
		bundleSupport,
		[]testutil.VideoUnitIndexFixture{
			{
				CoarseUnitID:       hardUnit,
				MentionCount:       1,
				SentenceCount:      1,
				FirstStartMs:       1_200,
				LastEndMs:          2_100,
				CoverageMs:         900,
				CoverageRatio:      0.05,
				SentenceIndexes:    []int32{0},
				EvidenceSpanRefs:   []testutil.EvidenceRefFixture{{SentenceIndex: 0, SpanIndex: 0}},
				SampleSurfaceForms: []string{"hard weak"},
			},
			{
				CoarseUnitID:       softUnitB,
				MentionCount:       3,
				SentenceCount:      1,
				FirstStartMs:       4_000,
				LastEndMs:          5_000,
				CoverageMs:         1_000,
				CoverageRatio:      0.06,
				SentenceIndexes:    []int32{2},
				EvidenceSpanRefs:   []testutil.EvidenceRefFixture{{SentenceIndex: 2, SpanIndex: 0}},
				SampleSurfaceForms: []string{"soft b"},
			},
		},
		[]testutil.SemanticSpanFixture{
			{SentenceIndex: 0, SpanIndex: 0, CoarseUnitID: &hardUnit, StartMs: 1_200, EndMs: 2_100, Text: "hard weak"},
			{SentenceIndex: 2, SpanIndex: 0, CoarseUnitID: &softUnitB, StartMs: 4_000, EndMs: 5_000, Text: "soft b"},
		},
		[]testutil.TranscriptSentenceFixture{
			{SentenceIndex: 0, Text: "hard weak", StartMs: 1_200, EndMs: 2_100},
			{SentenceIndex: 1, Text: "bridge", StartMs: 2_200, EndMs: 3_200},
			{SentenceIndex: 2, Text: "soft b", StartMs: 4_000, EndMs: 5_000},
		},
		110_000,
		0.72,
	))
	h.RefreshRecommendationViews(t)

	testutil.MustEnsureTarget(t, learning, userID,
		targetSpec(hardUnit, 0.95, "hard"),
		targetSpec(softUnitA, 0.70, "soft_a"),
		targetSpec(softUnitB, 0.68, "soft_b"),
	)

	now := time.Now().UTC()
	q4 := int16(4)
	q1 := int16(1)
	if _, err := learning.RecordEvents.Execute(context.Background(), learningdto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []learningdto.LearningEventInput{
			{CoarseUnitID: hardUnit, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: now.Add(-48 * time.Hour)},
			{CoarseUnitID: hardUnit, EventType: "review", SourceType: "quiz_session", Quality: &q1, OccurredAt: now.Add(-12 * time.Hour)},
			{CoarseUnitID: softUnitA, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: now.Add(-3 * time.Hour)},
			{CoarseUnitID: softUnitB, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: now.Add(-4 * time.Hour)},
		},
	}); err != nil {
		t.Fatalf("RecordLearningEvents.Execute(): %v", err)
	}

	response := testutil.MustRecommend(t, recommendation, userID, 3)
	if response.SelectorMode != "low_supply" {
		t.Fatalf("selector_mode = %q, want low_supply", response.SelectorMode)
	}
	if len(response.Videos) == 0 {
		t.Fatal("expected low-supply recommendation result")
	}

	coreCovered := false
	for _, video := range response.Videos {
		if containsUnit(video.CoveredHardReviewUnits, hardUnit) {
			coreCovered = true
			break
		}
	}
	if !coreCovered {
		t.Fatalf("hard-review unit %d was not preserved in low-supply result: %+v", hardUnit, response.Videos)
	}
}

func TestE2E_RecommendationNoDemandDoesNotMarkExtremeSparse(t *testing.T) {
	h := harness(t)
	recommendation := h.RecommendationUsecase()

	userID := h.NewUserID()
	h.SeedUser(t, userID)

	response := testutil.MustRecommend(t, recommendation, userID, 3)
	if response.SelectorMode != "normal" {
		t.Fatalf("selector_mode = %q, want normal", response.SelectorMode)
	}
	if !response.Underfilled {
		t.Fatalf("underfilled = false, want true")
	}
	if len(response.Videos) != 0 {
		t.Fatalf("videos = %#v, want empty", response.Videos)
	}
}
