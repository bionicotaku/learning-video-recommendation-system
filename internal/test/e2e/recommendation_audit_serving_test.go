//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	learningdto "learning-video-recommendation-system/internal/learningengine/application/dto"
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
			{CoarseUnitID: unitID, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: now.Add(-48 * time.Hour)},
			{CoarseUnitID: unitID, EventType: "review", SourceType: "quiz_session", Quality: &q2, OccurredAt: now.Add(-12 * time.Hour)},
		},
	}); err != nil {
		t.Fatalf("RecordLearningEvents.Execute(): %v", err)
	}

	response := testutil.MustRecommend(t, recommendation, userID, 1)
	if len(response.Videos) != 1 {
		t.Fatalf("expected exactly one video, got %d", len(response.Videos))
	}
	if response.Videos[0].BestEvidenceSentenceIndex == nil || response.Videos[0].BestEvidenceSpanIndex == nil || response.Videos[0].BestEvidenceStartMs == nil || response.Videos[0].BestEvidenceEndMs == nil {
		t.Fatalf("missing best evidence in response: %+v", response.Videos[0])
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

	auditSentence, auditSpan, auditStart, auditEnd := h.LoadAuditEvidence(t, response.RunID, 1)
	if *auditSentence != *response.Videos[0].BestEvidenceSentenceIndex ||
		*auditSpan != *response.Videos[0].BestEvidenceSpanIndex ||
		*auditStart != *response.Videos[0].BestEvidenceStartMs ||
		*auditEnd != *response.Videos[0].BestEvidenceEndMs {
		t.Fatalf("audit evidence mismatch: response=%+v audit=(%v,%v,%v,%v)", response.Videos[0], auditSentence, auditSpan, auditStart, auditEnd)
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
			{CoarseUnitID: unitID, EventType: "new_learn", SourceType: "quiz_session", Quality: &q4, OccurredAt: now.Add(-48 * time.Hour)},
			{CoarseUnitID: unitID, EventType: "review", SourceType: "quiz_session", Quality: &q2, OccurredAt: now.Add(-6 * time.Hour)},
		},
	}); err != nil {
		t.Fatalf("RecordLearningEvents.Execute(): %v", err)
	}

	first := testutil.MustRecommend(t, recommendation, userID, 1)
	if len(first.Videos) != 1 {
		t.Fatalf("expected 1 first-run video, got %d", len(first.Videos))
	}

	lastWatchedAt := time.Now().UTC()
	h.SeedVideoUserState(t, userID, first.Videos[0].VideoID, &lastWatchedAt, 5, 2, 0.90, 0.95)

	second := testutil.MustRecommend(t, recommendation, userID, 1)
	if len(second.Videos) != 1 {
		t.Fatalf("expected 1 second-run video, got %d", len(second.Videos))
	}
	if second.Videos[0].VideoID == first.Videos[0].VideoID {
		t.Fatalf("expected second run to avoid repeating top video after serving/watch penalty, got %s twice", second.Videos[0].VideoID)
	}

	if got := h.LoadVideoServingCount(t, userID, first.Videos[0].VideoID); got != 1 {
		t.Fatalf("first video served_count should remain 1, got %d", got)
	}
	if got := h.LoadVideoServingCount(t, userID, second.Videos[0].VideoID); got != 1 {
		t.Fatalf("second video served_count should be 1, got %d", got)
	}
}
