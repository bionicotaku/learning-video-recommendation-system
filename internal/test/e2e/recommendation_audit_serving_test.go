//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	"learning-video-recommendation-system/internal/test/e2e/testutil"
)

func TestE2E_RecommendationWritesAuditAndServingStateWithEvidence(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()
	recommendation := h.RecommendationUsecase()

	userID := h.NewUserID()
	unitID := h.NewUnitID()
	videoID := h.NewVideoID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, unitID)
	h.SeedCatalogVideo(t, singleUnitVideo(videoID, unitID, 1_000, 2_500, 0, "audit evidence", 90_000, 4, 0.10, 0.86))
	h.RefreshRecommendationViews(t)

	testutil.MustEnsureTarget(t, learning, userID, targetSpec(unitID, 0.95, "audit"))

	now := time.Now().UTC()
	q4 := int16(4)
	q2 := int16(2)
	if _, err := learning.RecordEvents.Execute(context.Background(), learningdto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []learningdto.LearningEventInput{
			{CoarseUnitID: unitID, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", SourceRefID: "audit-serving-1", ProgressQuality: &q4, OccurredAt: now.Add(-48 * time.Hour)},
			{CoarseUnitID: unitID, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", SourceRefID: "audit-serving-2", ProgressQuality: &q2, OccurredAt: now.Add(-12 * time.Hour)},
		},
	}); err != nil {
		t.Fatalf("RecordLearningEvents.Execute(): %v", err)
	}

	response := testutil.MustRecommend(t, recommendation, userID, 1)
	if len(response.Items) != 1 {
		t.Fatalf("expected exactly one video, got %d", len(response.Items))
	}
	if len(response.Items[0].LearningUnits) != 1 ||
		response.Items[0].LearningUnits[0].Evidence == nil ||
		response.Items[0].LearningUnits[0].Evidence.SentenceIndex == nil ||
		response.Items[0].LearningUnits[0].Evidence.SpanIndex == nil ||
		response.Items[0].LearningUnits[0].Evidence.StartMs == nil ||
		response.Items[0].LearningUnits[0].Evidence.EndMs == nil {
		t.Fatalf("missing learning unit evidence in response: %+v", response.Items[0])
	}

	if got := h.CountRecommendationRuns(t, userID); got != 1 {
		t.Fatalf("recommendation run count = %d, want 1", got)
	}
	if got := h.CountRecommendationItems(t, response.RunID); got != 1 {
		t.Fatalf("recommendation item count = %d, want 1", got)
	}
	if got := h.LoadUnitServingCount(t, userID, unitID); got != 1 {
		t.Fatalf("unit served_count = %d, want 1", got)
	}
	if got := h.LoadVideoServingCount(t, userID, videoID); got != 1 {
		t.Fatalf("video served_count = %d, want 1", got)
	}

	auditUnits := h.LoadAuditLearningUnits(t, response.RunID, 1)
	if len(auditUnits) != 1 {
		t.Fatalf("audit learning_units = %+v, want one unit", auditUnits)
	}
	responseEvidence := response.Items[0].LearningUnits[0].Evidence
	auditEvidence := auditUnits[0].Evidence
	if auditEvidence == nil ||
		auditEvidence.SentenceIndex == nil ||
		auditEvidence.SpanIndex == nil ||
		auditEvidence.StartMs == nil ||
		auditEvidence.EndMs == nil ||
		*auditEvidence.SentenceIndex != *responseEvidence.SentenceIndex ||
		*auditEvidence.SpanIndex != *responseEvidence.SpanIndex ||
		*auditEvidence.StartMs != *responseEvidence.StartMs ||
		*auditEvidence.EndMs != *responseEvidence.EndMs {
		t.Fatalf("audit learning unit evidence mismatch: response=%+v audit=%+v", response.Items[0].LearningUnits, auditUnits)
	}
}

func TestE2E_RecommendationSecondRunAppliesServingAndWatchedPenalty(t *testing.T) {
	h := harness(t)
	learning := h.LearningSuite()
	recommendation := h.RecommendationUsecase()

	userID := h.NewUserID()
	unitID := h.NewUnitID()
	videoA := h.NewVideoID()
	videoB := h.NewVideoID()
	h.SeedUser(t, userID)
	h.SeedCoarseUnits(t, unitID)

	h.SeedCatalogVideo(t, singleUnitVideo(videoA, unitID, 1_000, 2_700, 0, "video a", 90_000, 4, 0.10, 0.90))
	h.SeedCatalogVideo(t, singleUnitVideo(videoB, unitID, 1_000, 2_400, 0, "video b", 90_000, 3, 0.08, 0.84))
	h.RefreshRecommendationViews(t)

	testutil.MustEnsureTarget(t, learning, userID, targetSpec(unitID, 0.95, "repeat"))

	now := time.Now().UTC()
	q4 := int16(4)
	q2 := int16(2)
	if _, err := learning.RecordEvents.Execute(context.Background(), learningdto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []learningdto.LearningEventInput{
			{CoarseUnitID: unitID, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", SourceRefID: "audit-replay-1", ProgressQuality: &q4, OccurredAt: now.Add(-48 * time.Hour)},
			{CoarseUnitID: unitID, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", SourceRefID: "audit-replay-2", ProgressQuality: &q2, OccurredAt: now.Add(-6 * time.Hour)},
		},
	}); err != nil {
		t.Fatalf("RecordLearningEvents.Execute(): %v", err)
	}

	first := testutil.MustRecommend(t, recommendation, userID, 1)
	if len(first.Items) != 1 {
		t.Fatalf("expected 1 first-run video, got %d", len(first.Items))
	}

	lastWatchedAt := time.Now().UTC()
	h.SeedVideoUserState(t, userID, first.Items[0].VideoID, &lastWatchedAt, 5, 2, 81_000, 85_500, 180_000)

	second := testutil.MustRecommend(t, recommendation, userID, 1)
	if len(second.Items) != 1 {
		t.Fatalf("expected 1 second-run video, got %d", len(second.Items))
	}
	if second.Items[0].VideoID == first.Items[0].VideoID {
		t.Fatalf("expected second run to avoid repeating top video after serving/watch penalty, got %s twice", second.Items[0].VideoID)
	}

	if got := h.LoadVideoServingCount(t, userID, first.Items[0].VideoID); got != 1 {
		t.Fatalf("first video served_count should remain 1, got %d", got)
	}
	if got := h.LoadVideoServingCount(t, userID, second.Items[0].VideoID); got != 1 {
		t.Fatalf("second video served_count should be 1, got %d", got)
	}
}
