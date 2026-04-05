package scenario_test

import (
	"testing"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"
	"learning-video-recommendation-system/internal/recommendation/scheduler/test/integration/fixture"
)

func TestGenerateRecommendationsPrefersDueReviewWhenQuotaIsTight(t *testing.T) {
	ctx, pool := fixture.NewTestPool(t)

	userID, err := fixture.CreateTestUser(ctx, pool)
	if err != nil {
		t.Fatalf("CreateTestUser() error = %v", err)
	}
	unitIDs, err := fixture.CreateTestCoarseUnits(ctx, pool, 2)
	if err != nil {
		t.Fatalf("CreateTestCoarseUnits() error = %v", err)
	}
	t.Cleanup(func() {
		fixture.CleanupTestData(ctx, t, pool, userID, unitIDs)
	})

	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	nilTime := any(nil)
	nilText := any(nil)
	nilInt := any(nil)

	if err := fixture.InsertState(ctx, pool,
		userID, unitIDs[0], true, "lesson", "review-1", 0.7, "reviewing", 45.0, 0.5,
		nilTime, nilTime, nilTime, 0, 0, 0, 0, 0, 0, 0, nilInt, []int16{}, []bool{}, 2, 3.0, 2.5, now.Add(-2*time.Hour), nilText, now, now,
	); err != nil {
		t.Fatalf("insertState(review) error = %v", err)
	}
	if err := fixture.InsertState(ctx, pool,
		userID, unitIDs[1], true, "lesson", "new-1", 0.9, "new", 0.0, 0.0,
		nilTime, nilTime, nilTime, 0, 0, 0, 0, 0, 0, 0, nilInt, []int16{}, []bool{}, 0, 0.0, 2.5, nilTime, nilText, now, now,
	); err != nil {
		t.Fatalf("insertState(new) error = %v", err)
	}

	uc := fixture.NewGenerateUseCase(pool, sqlcgen.New(pool))
	result, err := uc.Execute(ctx, fixture.GenerateCmd(userID, 1, now))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(result.Batch.Items) != 1 {
		t.Fatalf("len(Batch.Items) = %d, want 1", len(result.Batch.Items))
	}
	if result.Batch.Items[0].RecommendType != enum.RecommendTypeReview {
		t.Fatalf("Batch.Items[0].RecommendType = %q, want %q", result.Batch.Items[0].RecommendType, enum.RecommendTypeReview)
	}
	if result.Batch.Items[0].CoarseUnitID != unitIDs[0] {
		t.Fatalf("Batch.Items[0].CoarseUnitID = %d, want %d", result.Batch.Items[0].CoarseUnitID, unitIDs[0])
	}
}

func TestGenerateRecommendationsSuppressesRecentlyRecommendedNew(t *testing.T) {
	ctx, pool := fixture.NewTestPool(t)

	userID, err := fixture.CreateTestUser(ctx, pool)
	if err != nil {
		t.Fatalf("CreateTestUser() error = %v", err)
	}
	unitIDs, err := fixture.CreateTestCoarseUnits(ctx, pool, 2)
	if err != nil {
		t.Fatalf("CreateTestCoarseUnits() error = %v", err)
	}
	t.Cleanup(func() {
		fixture.CleanupTestData(ctx, t, pool, userID, unitIDs)
	})

	now := time.Date(2026, 4, 10, 14, 0, 0, 0, time.UTC)
	nilTime := any(nil)
	nilText := any(nil)
	nilInt := any(nil)

	if err := fixture.InsertState(ctx, pool,
		userID, unitIDs[0], true, "lesson", "new-old", 0.8, "new", 0.0, 0.0,
		nilTime, nilTime, nilTime, 0, 0, 0, 0, 0, 0, 0, nilInt, []int16{}, []bool{}, 0, 0.0, 2.5, nilTime, nilText, now, now,
	); err != nil {
		t.Fatalf("insertState(old) error = %v", err)
	}
	if err := fixture.InsertState(ctx, pool,
		userID, unitIDs[1], true, "lesson", "new-recent", 0.8, "new", 0.0, 0.0,
		nilTime, nilTime, nilTime, 0, 0, 0, 0, 0, 0, 0, nilInt, []int16{}, []bool{}, 0, 0.0, 2.5, nilTime, nilText, now, now,
	); err != nil {
		t.Fatalf("insertState(recent) error = %v", err)
	}
	if err := fixture.InsertServingState(ctx, pool, userID, unitIDs[0], now.Add(-25*time.Hour)); err != nil {
		t.Fatalf("insertServingState(old) error = %v", err)
	}
	if err := fixture.InsertServingState(ctx, pool, userID, unitIDs[1], now.Add(-1*time.Hour)); err != nil {
		t.Fatalf("insertServingState(recent) error = %v", err)
	}

	uc := fixture.NewGenerateUseCase(pool, sqlcgen.New(pool))
	result, err := uc.Execute(ctx, fixture.GenerateCmd(userID, 1, now))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(result.Batch.Items) != 1 {
		t.Fatalf("len(Batch.Items) = %d, want 1", len(result.Batch.Items))
	}
	if result.Batch.Items[0].RecommendType != enum.RecommendTypeNew {
		t.Fatalf("Batch.Items[0].RecommendType = %q, want %q", result.Batch.Items[0].RecommendType, enum.RecommendTypeNew)
	}
	if result.Batch.Items[0].CoarseUnitID != unitIDs[0] {
		t.Fatalf("Batch.Items[0].CoarseUnitID = %d, want %d", result.Batch.Items[0].CoarseUnitID, unitIDs[0])
	}
}
