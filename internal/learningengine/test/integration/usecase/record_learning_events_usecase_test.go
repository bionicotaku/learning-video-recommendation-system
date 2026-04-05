package usecase_test

import (
	"testing"
	"time"

	"learning-video-recommendation-system/internal/learningengine/application/command"
	"learning-video-recommendation-system/internal/learningengine/domain/enum"
	repopkg "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/sqlcgen"
	"learning-video-recommendation-system/internal/learningengine/test/integration/fixture"
)

func TestRecordLearningEventsUseCase(t *testing.T) {
	ctx, pool := fixture.NewTestPool(t)

	userID, err := fixture.CreateTestUser(ctx, pool)
	if err != nil {
		t.Fatalf("CreateTestUser() error = %v", err)
	}
	unitIDs, err := fixture.CreateTestCoarseUnits(ctx, pool, 1)
	if err != nil {
		t.Fatalf("CreateTestCoarseUnits() error = %v", err)
	}
	t.Cleanup(func() {
		fixture.CleanupTestData(ctx, t, pool, userID, unitIDs)
	})

	baseQuerier := sqlcgen.New(pool)
	stateRepo := repopkg.NewUserUnitStateRepository(baseQuerier)
	eventRepo := repopkg.NewUnitLearningEventRepository(baseQuerier)
	uc := fixture.NewRecordEventsUseCase(pool, baseQuerier)

	correct := true
	quality := 4
	occurredAt := time.Date(2026, 4, 8, 15, 0, 0, 0, time.UTC)

	result, err := uc.Execute(ctx, command.RecordLearningEventsCommand{
		UserID: userID,
		Events: []command.LearningEventInput{
			fixture.NewLearnInput(unitIDs[0], &correct, &quality, occurredAt, "record-events"),
		},
		IdempotencyKey: "integration-success",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.AcceptedCount != 1 {
		t.Fatalf("AcceptedCount = %d, want 1", result.AcceptedCount)
	}
	if len(result.UpdatedUnits) != 1 || result.UpdatedUnits[0] != unitIDs[0] {
		t.Fatalf("UpdatedUnits = %v, want [%d]", result.UpdatedUnits, unitIDs[0])
	}

	events, err := eventRepo.ListByUserOrdered(ctx, userID)
	if err != nil {
		t.Fatalf("ListByUserOrdered() error = %v", err)
	}
	events = fixture.FilterEventsByUnit(events, userID, unitIDs[0])
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}

	state, err := stateRepo.GetByUserAndUnit(ctx, userID, unitIDs[0])
	if err != nil {
		t.Fatalf("GetByUserAndUnit() error = %v", err)
	}
	if state == nil {
		t.Fatal("state = nil, want value")
	}
	if state.Status != enum.UnitStatusLearning {
		t.Fatalf("state.Status = %q, want %q", state.Status, enum.UnitStatusLearning)
	}
}
