//go:build e2e

package e2e

import (
	"testing"
	"time"

	"learning-video-recommendation-system/internal/test/e2e/testutil"
)

func TestE2E_RecommendationReadModelVisibility_RefreshGatesNewVideos(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()
	recommendation := h.RecommendationUsecase()

	userID := h.NewUserID()
	unitID := h.NewUnitID()
	videoID := h.NewVideoID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, unitID)
	testutil.MustEnsureTarget(t, learning, userID, targetSpec(unitID, 0.95, "refresh_gate"))

	h.SeedCatalogVideo(t, strongSupplyVideo(videoID, unitID, 1_000, 2_000, 0, "refresh-gated", 90_000))

	beforeRefresh := mustRecommendN(t, recommendation, userID, 1)
	if len(beforeRefresh.Videos) != 0 {
		t.Fatalf("before refresh expected empty result, got %v", videoIDs(beforeRefresh.Videos))
	}

	h.RefreshRecommendationViews(t)

	afterRefresh := mustRecommendN(t, recommendation, userID, 1)
	assertContainsVideo(t, afterRefresh.Videos, videoID)
}

func TestE2E_RecommendationReadModelVisibility_FiltersInactiveHiddenAndFutureVideos(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()
	recommendation := h.RecommendationUsecase()

	userID := h.NewUserID()
	unitID := h.NewUnitID()
	visibleVideo := h.NewVideoID()
	hiddenVideoID := h.NewVideoID()
	inactiveVideoID := h.NewVideoID()
	futureVideoID := h.NewVideoID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, unitID)

	h.SeedCatalogVideo(t, strongSupplyVideo(visibleVideo, unitID, 1_000, 2_100, 0, "visible", 90_000))
	h.SeedCatalogVideo(t, hiddenVideo(strongSupplyVideo(hiddenVideoID, unitID, 3_000, 4_100, 2, "hidden", 90_000)))
	h.SeedCatalogVideo(t, inactiveVideo(strongSupplyVideo(inactiveVideoID, unitID, 5_000, 6_100, 4, "inactive", 90_000)))
	h.SeedCatalogVideo(t, futurePublishVideo(strongSupplyVideo(futureVideoID, unitID, 7_000, 8_100, 6, "future", 90_000), mustTimeAdd(time.Now().UTC(), 24*time.Hour)))
	h.RefreshRecommendationViews(t)

	testutil.MustEnsureTarget(t, learning, userID, targetSpec(unitID, 0.95, "visibility"))

	response := mustRecommendN(t, recommendation, userID, 3)
	assertContainsVideo(t, response.Videos, visibleVideo)
	assertNotContainsVideo(t, response.Videos, hiddenVideoID)
	assertNotContainsVideo(t, response.Videos, inactiveVideoID)
	assertNotContainsVideo(t, response.Videos, futureVideoID)
}

func TestE2E_RecommendationReadModelVisibility_InventorySupplyGradeContract(t *testing.T) {
	h := harness(t)

	noneUnit := h.NewUnitID()
	weakUnit := h.NewUnitID()
	okUnit := h.NewUnitID()
	strongUnit := h.NewUnitID()
	h.SeedCoarseUnits(t, noneUnit, weakUnit, okUnit, strongUnit)

	h.SeedCatalogVideo(t, weakSupplyVideo(h.NewVideoID(), weakUnit, 1_000, 2_000, 0, "weak", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(h.NewVideoID(), okUnit, 3_000, 4_000, 2, "ok-a", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(h.NewVideoID(), okUnit, 5_000, 6_000, 4, "ok-b", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(h.NewVideoID(), strongUnit, 7_000, 8_000, 6, "strong-a", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(h.NewVideoID(), strongUnit, 9_000, 10_000, 8, "strong-b", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(h.NewVideoID(), strongUnit, 11_000, 12_000, 10, "strong-c", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(h.NewVideoID(), strongUnit, 13_000, 14_000, 12, "strong-d", 90_000))
	h.RefreshRecommendationViews(t)

	if got := h.LoadSupplyGrade(t, noneUnit); got != "none" {
		t.Fatalf("none unit supply grade = %q, want none", got)
	}
	if got := h.LoadSupplyGrade(t, weakUnit); got != "weak" {
		t.Fatalf("weak unit supply grade = %q, want weak", got)
	}
	if got := h.LoadSupplyGrade(t, okUnit); got != "ok" {
		t.Fatalf("ok unit supply grade = %q, want ok", got)
	}
	if got := h.LoadSupplyGrade(t, strongUnit); got != "strong" {
		t.Fatalf("strong unit supply grade = %q, want strong", got)
	}
}
