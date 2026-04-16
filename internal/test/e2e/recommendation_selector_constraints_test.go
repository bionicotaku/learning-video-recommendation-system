//go:build e2e

package e2e

import (
	"testing"
	"time"

	learningdto "learning-video-recommendation-system/internal/learningengine/application/dto"
	"learning-video-recommendation-system/internal/test/e2e/testutil"
)

func TestE2E_RecommendationSelectorMarksExtremeSparseWhenDemandUnderfills(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()
	recommendation := h.RecommendationUsecase()

	userID := h.NewUserID()
	unitID := h.NewUnitID()
	videoID := h.NewVideoID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, unitID)
	h.SeedCatalogVideo(t, strongSupplyVideo(videoID, unitID, 1_000, 2_000, 0, "sparse-hard", 90_000))
	h.RefreshRecommendationViews(t)

	testutil.MustEnsureTarget(t, learning, userID, targetSpec(unitID, 0.95, "hard_sparse"))

	now := time.Now().UTC()
	q4 := int16(4)
	mustRecordEvents(t, learning, userID,
		learningdto.LearningEventInput{CoarseUnitID: unitID, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: mustTimeAdd(now, -48*time.Hour)},
	)

	response := mustRecommendN(t, recommendation, userID, 3)
	assertSelectorMode(t, response, "extreme_sparse")
	if !response.Underfilled {
		t.Fatalf("underfilled = false, want true")
	}
	if len(response.Videos) != 1 {
		t.Fatalf("len(videos) = %d, want 1", len(response.Videos))
	}
}

func TestE2E_RecommendationSelectorRespectsSameUnitMax(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()
	recommendation := h.RecommendationUsecase()

	userID := h.NewUserID()
	heavyUnit := h.NewUnitID()
	otherHardUnit := h.NewUnitID()
	newUnit := h.NewUnitID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, heavyUnit, otherHardUnit, newUnit)

	videoA := h.NewVideoID()
	videoB := h.NewVideoID()
	videoC := h.NewVideoID()
	videoD := h.NewVideoID()
	videoE := h.NewVideoID()
	h.SeedCatalogVideo(t, strongSupplyVideo(videoA, heavyUnit, 1_000, 2_100, 0, "heavy-a", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(videoB, heavyUnit, 3_000, 4_100, 2, "heavy-b", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(videoC, heavyUnit, 5_000, 6_100, 4, "heavy-c", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(videoD, otherHardUnit, 7_000, 8_100, 6, "other-hard", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(videoE, newUnit, 9_000, 10_100, 8, "new-core", 90_000))
	h.RefreshRecommendationViews(t)

	testutil.MustEnsureTarget(t, learning, userID,
		targetSpec(heavyUnit, 0.95, "heavy"),
		targetSpec(otherHardUnit, 0.90, "other_hard"),
		targetSpec(newUnit, 0.85, "new"),
	)

	now := time.Now().UTC()
	q4 := int16(4)
	mustRecordEvents(t, learning, userID,
		learningdto.LearningEventInput{CoarseUnitID: heavyUnit, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: mustTimeAdd(now, -48*time.Hour)},
		learningdto.LearningEventInput{CoarseUnitID: otherHardUnit, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: mustTimeAdd(now, -48*time.Hour)},
	)

	response := mustRecommendN(t, recommendation, userID, 4)
	items := h.LoadRecommendationItems(t, response.RunID)
	if got := countDominantUnit(items, heavyUnit); got > 2 {
		t.Fatalf("dominant selections for unit %d = %d, want <= 2", heavyUnit, got)
	}
}

func TestE2E_RecommendationSelectorRespectsFallbackMaxAndCoreDominantMin(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()
	recommendation := h.RecommendationUsecase()

	userID := h.NewUserID()
	hardUnit := h.NewUnitID()
	newExactA := h.NewUnitID()
	newExactB := h.NewUnitID()
	newFallback := h.NewUnitID()
	softUnit := h.NewUnitID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, hardUnit, newExactA, newExactB, newFallback, softUnit)

	h.SeedCatalogVideo(t, strongSupplyVideo(h.NewVideoID(), hardUnit, 1_000, 2_100, 0, "hard", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(h.NewVideoID(), newExactA, 3_000, 4_100, 2, "new-a", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(h.NewVideoID(), newExactB, 5_000, 6_100, 4, "new-b", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(h.NewVideoID(), newFallback, 7_000, 8_100, 6, "new-fallback", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(h.NewVideoID(), softUnit, 9_000, 10_100, 8, "soft", 90_000))
	h.RefreshRecommendationViews(t)

	testutil.MustEnsureTarget(t, learning, userID,
		targetSpec(hardUnit, 0.95, "hard"),
		targetSpec(newExactA, 0.90, "new_a"),
		targetSpec(newExactB, 0.89, "new_b"),
		targetSpec(newFallback, 0.60, "new_fallback"),
		targetSpec(softUnit, 0.55, "soft"),
	)

	now := time.Now().UTC()
	q4 := int16(4)
	mustRecordEvents(t, learning, userID,
		learningdto.LearningEventInput{CoarseUnitID: hardUnit, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: mustTimeAdd(now, -48*time.Hour)},
		learningdto.LearningEventInput{CoarseUnitID: softUnit, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: mustTimeAdd(now, -12*time.Hour)},
	)

	response := mustRecommendN(t, recommendation, userID, 5)
	items := h.LoadRecommendationItems(t, response.RunID)
	if got := countPrimaryLane(items, "quality_fallback"); got != 1 {
		t.Fatalf("quality_fallback count = %d, want 1", got)
	}
	if got := countCoreDominant(items); got < 2 {
		t.Fatalf("core dominant selections = %d, want >= 2", got)
	}
}

func TestE2E_RecommendationSelectorRespectsFutureLikeMaxInLowSupply(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()
	recommendation := h.RecommendationUsecase()

	userID := h.NewUserID()
	hardUnit := h.NewUnitID()
	newUnit := h.NewUnitID()
	softA := h.NewUnitID()
	softB := h.NewUnitID()
	softC := h.NewUnitID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, hardUnit, newUnit, softA, softB, softC)

	h.SeedCatalogVideo(t, weakSupplyVideo(h.NewVideoID(), hardUnit, 1_000, 2_000, 0, "weak-hard", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(h.NewVideoID(), newUnit, 2_200, 3_200, 1, "new-fill", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(h.NewVideoID(), softA, 3_000, 4_200, 2, "soft-a", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(h.NewVideoID(), softB, 5_000, 6_200, 4, "soft-b", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(h.NewVideoID(), softC, 7_000, 8_200, 6, "soft-c", 90_000))
	h.RefreshRecommendationViews(t)

	testutil.MustEnsureTarget(t, learning, userID,
		targetSpec(hardUnit, 0.95, "hard"),
		targetSpec(newUnit, 0.90, "new"),
		targetSpec(softA, 0.75, "soft_a"),
		targetSpec(softB, 0.74, "soft_b"),
		targetSpec(softC, 0.73, "soft_c"),
	)

	now := time.Now().UTC()
	q4 := int16(4)
	q1 := int16(1)
	mustRecordEvents(t, learning, userID,
		learningdto.LearningEventInput{CoarseUnitID: hardUnit, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: mustTimeAdd(now, -48*time.Hour)},
		learningdto.LearningEventInput{CoarseUnitID: hardUnit, EventType: "review", SourceType: "quiz_session", Quality: &q1, OccurredAt: mustTimeAdd(now, -12*time.Hour)},
		learningdto.LearningEventInput{CoarseUnitID: softA, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: mustTimeAdd(now, -12*time.Hour)},
		learningdto.LearningEventInput{CoarseUnitID: softB, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: mustTimeAdd(now, -12*time.Hour)},
		learningdto.LearningEventInput{CoarseUnitID: softC, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: mustTimeAdd(now, -12*time.Hour)},
	)

	response := mustRecommendN(t, recommendation, userID, 4)
	assertSelectorMode(t, response, "low_supply")
	items := h.LoadRecommendationItems(t, response.RunID)
	if got := countFutureLike(items); got > 2 {
		t.Fatalf("future-like dominant selections = %d, want <= 2", got)
	}
}
