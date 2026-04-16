//go:build e2e

package e2e

import (
	"testing"
	"time"

	learningdto "learning-video-recommendation-system/internal/learningengine/application/dto"
	"learning-video-recommendation-system/internal/test/e2e/testutil"
)

func TestE2E_MultiUserRecommendationServingIsolation(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()
	recommendation := h.RecommendationUsecase()

	userA := h.NewUserID()
	userB := h.NewUserID()
	unitA := h.NewUnitID()
	unitB := h.NewUnitID()
	videoA := h.NewVideoID()
	videoB := h.NewVideoID()
	h.SeedUser(t, userA)
	h.SeedUser(t, userB)
	h.SeedCoarseUnits(t, unitA, unitB)
	h.SeedCatalogVideo(t, strongSupplyVideo(videoA, unitA, 1_000, 2_200, 0, "user-a", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(videoB, unitB, 3_000, 4_200, 2, "user-b", 90_000))
	h.RefreshRecommendationViews(t)

	testutil.MustEnsureTarget(t, learning, userA, targetSpec(unitA, 0.95, "ua"))
	testutil.MustEnsureTarget(t, learning, userB, targetSpec(unitB, 0.95, "ub"))

	responseA := mustRecommendN(t, recommendation, userA, 1)
	responseB := mustRecommendN(t, recommendation, userB, 1)
	assertContainsVideo(t, responseA.Videos, videoA)
	assertNotContainsVideo(t, responseA.Videos, videoB)
	assertContainsVideo(t, responseB.Videos, videoB)
	assertNotContainsVideo(t, responseB.Videos, videoA)

	if got := h.LoadVideoServingCount(t, userA, videoA); got != 1 {
		t.Fatalf("userA videoA served_count = %d, want 1", got)
	}
	if got := h.LoadVideoServingCount(t, userB, videoB); got != 1 {
		t.Fatalf("userB videoB served_count = %d, want 1", got)
	}
}

func TestE2E_MultiUserReplayDoesNotAffectOtherUsersRecommendation(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()
	recommendation := h.RecommendationUsecase()

	userA := h.NewUserID()
	userB := h.NewUserID()
	unitA := h.NewUnitID()
	unitB := h.NewUnitID()
	videoA := h.NewVideoID()
	videoB := h.NewVideoID()
	h.SeedUser(t, userA)
	h.SeedUser(t, userB)
	h.SeedCoarseUnits(t, unitA, unitB)
	h.SeedCatalogVideo(t, strongSupplyVideo(videoA, unitA, 1_000, 2_200, 0, "replay-a", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(videoB, unitB, 3_000, 4_200, 2, "replay-b", 90_000))
	h.RefreshRecommendationViews(t)

	testutil.MustEnsureTarget(t, learning, userA, targetSpec(unitA, 0.95, "ua"))
	testutil.MustEnsureTarget(t, learning, userB, targetSpec(unitB, 0.95, "ub"))

	now := time.Now().UTC()
	q4 := int16(4)
	mustRecordEvents(t, learning, userA, learningdto.LearningEventInput{
		CoarseUnitID: unitA, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: mustTimeAdd(now, -48*time.Hour),
	})

	beforeReplayB := mustRecommendN(t, recommendation, userB, 1)
	mustReplay(t, learning, userA)
	afterReplayB := mustRecommendN(t, recommendation, userB, 1)

	assertContainsVideo(t, beforeReplayB.Videos, videoB)
	assertContainsVideo(t, afterReplayB.Videos, videoB)
	if got, want := videoIDs(afterReplayB.Videos), videoIDs(beforeReplayB.Videos); len(got) != len(want) {
		t.Fatalf("userB recommendation drifted after userA replay: before=%v after=%v", want, got)
	} else {
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("userB recommendation drifted after userA replay: before=%v after=%v", want, got)
			}
		}
	}
}
