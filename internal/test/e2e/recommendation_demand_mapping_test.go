//go:build e2e

package e2e

import (
	"encoding/json"
	"testing"
	"time"

	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	"learning-video-recommendation-system/internal/test/e2e/testutil"
)

func TestE2E_RecommendationDemandMapping_MixedBucketsFromLearningStates(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()
	recommendation := h.RecommendationUsecaseWithoutFill()

	userID := h.NewUserID()
	hardUnit := h.NewUnitID()
	newUnit := h.NewUnitID()
	softMasteryUnit := h.NewUnitID()
	softQualityUnit := h.NewUnitID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, hardUnit, newUnit, softMasteryUnit, softQualityUnit)

	hardVideo := h.NewVideoID()
	hardSupportVideo := h.NewVideoID()
	newVideo := h.NewVideoID()
	softMasteryVideo := h.NewVideoID()
	softQualityVideo := h.NewVideoID()
	h.SeedCatalogVideo(t, strongSupplyVideo(hardVideo, hardUnit, 1_000, 2_200, 0, "hard-bucket", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(hardSupportVideo, newUnit, 2_400, 3_300, 1, "new-bucket-support", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(newVideo, newUnit, 3_000, 4_100, 2, "new-bucket", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(softMasteryVideo, softMasteryUnit, 5_000, 6_300, 4, "soft-mastery", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(softQualityVideo, softQualityUnit, 7_000, 8_400, 6, "soft-quality", 90_000))
	h.RefreshRecommendationViews(t)

	testutil.MustEnsureTarget(t, learning, userID,
		targetSpec(hardUnit, 0.95, "hard"),
		targetSpec(newUnit, 0.90, "new"),
		targetSpec(softMasteryUnit, 0.75, "soft_mastery"),
		targetSpec(softQualityUnit, 0.72, "soft_quality"),
	)

	now := time.Now().UTC()
	q4 := int16(4)
	q3 := int16(3)
	mustRecordEvents(t, learning, userID,
		learningdto.LearningEventInput{CoarseUnitID: hardUnit, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", ProgressQuality: &q4, OccurredAt: mustTimeAdd(now, -48*time.Hour)},
		learningdto.LearningEventInput{CoarseUnitID: softMasteryUnit, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", ProgressQuality: &q4, OccurredAt: mustTimeAdd(now, -12*time.Hour)},
		learningdto.LearningEventInput{CoarseUnitID: softQualityUnit, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", ProgressQuality: &q4, OccurredAt: mustTimeAdd(now, -48*time.Hour)},
		learningdto.LearningEventInput{CoarseUnitID: softQualityUnit, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", ProgressQuality: &q3, OccurredAt: mustTimeAdd(now, -1*time.Hour)},
	)

	response := mustRecommendN(t, recommendation, userID, 4)
	assertSelectorMode(t, h, response, "low_supply")

	snapshot := decodeDemandSnapshot(t, h.LoadPlannerSnapshot(t, response.RunID))
	assertDemandBucketContains(t, snapshot.HardReview, hardUnit, "hard_review")
	assertDemandBucketContains(t, snapshot.NewNow, newUnit, "new_now")
	assertDemandBucketContains(t, snapshot.SoftReview, softQualityUnit, "soft_review")
}

func TestE2E_RecommendationDemandMapping_NewTargetWithoutSupplyMarksExtremeSparse(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()
	recommendation := h.RecommendationUsecaseWithoutFill()

	userID := h.NewUserID()
	unitID := h.NewUnitID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, unitID)

	testutil.MustEnsureTarget(t, learning, userID, targetSpec(unitID, 0.95, "no_supply_new"))

	response := mustRecommendN(t, recommendation, userID, 1)
	assertSelectorMode(t, h, response, "extreme_sparse")
	if !h.LoadRecommendationRun(t, response.RunID).Underfilled {
		t.Fatalf("underfilled = false, want true")
	}
	if len(response.Items) != 0 {
		t.Fatalf("items = %#v, want empty", videoIDs(response.Items))
	}
}

func TestE2E_RecommendationDemandMapping_SuspendedInactiveAndNonTargetUnitsAreExcluded(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()
	recommendation := h.RecommendationUsecaseWithoutFill()

	userID := h.NewUserID()
	activeUnit := h.NewUnitID()
	suspendedUnit := h.NewUnitID()
	inactiveUnit := h.NewUnitID()
	nonTargetUnit := h.NewUnitID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, activeUnit, suspendedUnit, inactiveUnit, nonTargetUnit)

	activeVideo := h.NewVideoID()
	suspendedVideo := h.NewVideoID()
	inactiveVideoID := h.NewVideoID()
	nonTargetVideo := h.NewVideoID()
	h.SeedCatalogVideo(t, strongSupplyVideo(activeVideo, activeUnit, 1_000, 2_200, 0, "active-target", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(suspendedVideo, suspendedUnit, 3_000, 4_200, 2, "suspended-target", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(inactiveVideoID, inactiveUnit, 5_000, 6_200, 4, "inactive-target", 90_000))
	h.SeedCatalogVideo(t, strongSupplyVideo(nonTargetVideo, nonTargetUnit, 7_000, 8_300, 6, "non-target", 90_000))
	h.RefreshRecommendationViews(t)

	testutil.MustEnsureTarget(t, learning, userID,
		targetSpec(activeUnit, 0.95, "active"),
		targetSpec(suspendedUnit, 0.90, "suspended"),
		targetSpec(inactiveUnit, 0.85, "inactive"),
	)

	now := time.Now().UTC()
	q4 := int16(4)
	mustRecordEvents(t, learning, userID,
		learningdto.LearningEventInput{CoarseUnitID: nonTargetUnit, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", ProgressQuality: &q4, OccurredAt: mustTimeAdd(now, -12*time.Hour)},
	)

	if _, err := learning.SuspendTargetUnit.Execute(ctx(), learningdto.SuspendTargetUnitRequest{
		UserID:          userID,
		CoarseUnitID:    suspendedUnit,
		SuspendedReason: "paused",
	}); err != nil {
		t.Fatalf("SuspendTargetUnit.Execute(): %v", err)
	}
	if _, err := learning.SetTargetInactive.Execute(ctx(), learningdto.SetTargetInactiveRequest{
		UserID:       userID,
		CoarseUnitID: inactiveUnit,
	}); err != nil {
		t.Fatalf("SetTargetInactive.Execute(): %v", err)
	}

	response := mustRecommendN(t, recommendation, userID, 4)
	assertContainsVideo(t, response.Items, activeVideo)
	assertNotContainsVideo(t, response.Items, suspendedVideo)
	assertNotContainsVideo(t, response.Items, inactiveVideoID)
	assertNotContainsVideo(t, response.Items, nonTargetVideo)
}

type demandSnapshot struct {
	HardReview []demandSnapshotUnit
	NewNow     []demandSnapshotUnit
	SoftReview []demandSnapshotUnit
}

type demandSnapshotUnit struct {
	UnitID int64
}

func decodeDemandSnapshot(t *testing.T, payload []byte) demandSnapshot {
	t.Helper()
	var snapshot demandSnapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		t.Fatalf("decode planner snapshot: %v", err)
	}
	return snapshot
}

func assertDemandBucketContains(t *testing.T, units []demandSnapshotUnit, unitID int64, bucket string) {
	t.Helper()
	for _, unit := range units {
		if unit.UnitID == unitID {
			return
		}
	}
	t.Fatalf("%s bucket does not contain unit %d: %+v", bucket, unitID, units)
}
