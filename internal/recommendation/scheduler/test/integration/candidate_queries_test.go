package integration

import (
	"testing"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"
	repopkg "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"

	"github.com/google/uuid"
)

func TestLearningStateSnapshotReadRepositoryCandidateQueries(t *testing.T) {
	ctx, pool := newTestPool(t)

	userID, err := createTestUser(ctx, pool)
	if err != nil {
		t.Fatalf("createTestUser() error = %v", err)
	}
	unitIDs, err := createTestCoarseUnits(ctx, pool, 3)
	if err != nil {
		t.Fatalf("createTestCoarseUnits() error = %v", err)
	}
	t.Cleanup(func() {
		cleanupTestData(ctx, t, pool, userID, unitIDs)
	})

	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	nilTime := any(nil)
	nilText := any(nil)
	nilInt := any(nil)

	if err := insertState(ctx, pool,
		userID, unitIDs[0], true, "lesson", "l-1", 0.9, "reviewing", 40.0, 0.4,
		nilTime, nilTime, nilTime, 0, 0, 0, 0, 0, 0, 0, nilInt, []int16{}, []bool{}, 2, 3.0, 2.5, now.Add(-1*time.Hour), nilText, now, now,
	); err != nil {
		t.Fatalf("insertState(review) error = %v", err)
	}
	if err := insertState(ctx, pool,
		userID, unitIDs[1], true, "lesson", "l-2", 0.8, "new", 0.0, 0.0,
		nilTime, nilTime, nilTime, 0, 0, 0, 0, 0, 0, 0, nilInt, []int16{}, []bool{}, 0, 0.0, 2.5, nilTime, nilText, now, now,
	); err != nil {
		t.Fatalf("insertState(new) error = %v", err)
	}
	if err := insertState(ctx, pool,
		userID, unitIDs[2], true, "lesson", "l-3", 0.7, "learning", 10.0, 0.1,
		nilTime, nilTime, nilTime, 0, 0, 0, 0, 0, 0, 0, nilInt, []int16{}, []bool{}, 1, 1.0, 2.5, now.Add(2*time.Hour), nilText, now, now,
	); err != nil {
		t.Fatalf("insertState(future review) error = %v", err)
	}

	recommendedAt := now.Add(-8 * time.Hour)
	if err := insertServingState(ctx, pool, userID, unitIDs[0], recommendedAt); err != nil {
		t.Fatalf("insertServingState() error = %v", err)
	}

	repo := repopkg.NewLearningStateSnapshotReadRepository(sqlcgen.New(pool))

	reviewCandidates, err := repo.FindDueReviewCandidates(ctx, userID, now)
	if err != nil {
		t.Fatalf("FindDueReviewCandidates() error = %v", err)
	}
	if len(reviewCandidates) != 1 {
		t.Fatalf("len(reviewCandidates) = %d, want 1", len(reviewCandidates))
	}
	if reviewCandidates[0].State.CoarseUnitID != unitIDs[0] {
		t.Fatalf("review candidate coarseUnitID = %d, want %d", reviewCandidates[0].State.CoarseUnitID, unitIDs[0])
	}
	if reviewCandidates[0].Serving.LastRecommendedAt == nil || !reviewCandidates[0].Serving.LastRecommendedAt.Equal(recommendedAt) {
		t.Fatalf("review candidate lastRecommendedAt = %v, want %v", reviewCandidates[0].Serving.LastRecommendedAt, recommendedAt)
	}
	if reviewCandidates[0].Unit.Kind != enum.UnitKindWord && reviewCandidates[0].Unit.Kind != enum.UnitKindPhrase && reviewCandidates[0].Unit.Kind != enum.UnitKindGrammar {
		t.Fatalf("review candidate kind = %q, want supported kind", reviewCandidates[0].Unit.Kind)
	}

	newCandidates, err := repo.FindNewCandidates(ctx, userID)
	if err != nil {
		t.Fatalf("FindNewCandidates() error = %v", err)
	}
	if len(newCandidates) != 1 {
		t.Fatalf("len(newCandidates) = %d, want 1", len(newCandidates))
	}
	if newCandidates[0].State.CoarseUnitID != unitIDs[1] {
		t.Fatalf("new candidate coarseUnitID = %d, want %d", newCandidates[0].State.CoarseUnitID, unitIDs[1])
	}
	if newCandidates[0].Serving.LastRecommendedAt != nil {
		t.Fatalf("new candidate lastRecommendedAt = %v, want nil", newCandidates[0].Serving.LastRecommendedAt)
	}

	_ = uuid.Nil
}
