package integration

import (
	"testing"
	"time"

	"learning-video-recommendation-system/internal/learningengine/application/command"
	"learning-video-recommendation-system/internal/learningengine/domain/enum"
	repopkg "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/sqlcgen"
)

func TestRecordLearningEventsUseCase(t *testing.T) {
	ctx, pool := newTestPool(t)

	userID, err := createTestUser(ctx, pool)
	if err != nil {
		t.Fatalf("createTestUser() error = %v", err)
	}
	unitIDs, err := createTestCoarseUnits(ctx, pool, 1)
	if err != nil {
		t.Fatalf("createTestCoarseUnits() error = %v", err)
	}
	t.Cleanup(func() {
		cleanupTestData(ctx, t, pool, userID, unitIDs)
	})

	baseQuerier := sqlcgen.New(pool)
	stateRepo := repopkg.NewUserUnitStateRepository(baseQuerier)
	eventRepo := repopkg.NewUnitLearningEventRepository(baseQuerier)
	uc := newRecordEventsUseCase(pool, baseQuerier)

	correct := true
	quality := 4
	occurredAt := time.Date(2026, 4, 8, 15, 0, 0, 0, time.UTC)

	result, err := uc.Execute(ctx, command.RecordLearningEventsCommand{
		UserID: userID,
		Events: []command.LearningEventInput{
			newLearnInput(unitIDs[0], &correct, &quality, occurredAt, "record-events"),
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
	events = filterEventsByUnit(events, userID, unitIDs[0])
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
