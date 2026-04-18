//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/learningengine/domain/model"
	persistrepo "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/repository"
)

func TestUnitLearningEventRepositoryAppendAndList(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)

	repo := persistrepo.NewUnitLearningEventRepository(db.Pool)
	q4 := int16(4)
	events := []model.LearningEvent{
		{
			UserID:       userID,
			CoarseUnitID: 101,
			EventType:    "new_learn",
			SourceType:   "quiz_session",
			Quality:      &q4,
			Metadata:     []byte("{}"),
			OccurredAt:   time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
		},
	}

	if err := repo.Append(context.Background(), events); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	recorded, err := repo.ListByUserOrdered(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListByUserOrdered() error = %v", err)
	}
	if len(recorded) != 1 {
		t.Fatalf("recorded len = %d, want 1", len(recorded))
	}
	if recorded[0].EventType != "new_learn" {
		t.Fatalf("event_type = %q, want new_learn", recorded[0].EventType)
	}
}

func TestUserUnitStateRepositoryUpsertListAndDelete(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)

	repo := persistrepo.NewUserUnitStateRepository(db.Pool)
	state := &model.UserUnitState{
		UserID:            userID,
		CoarseUnitID:      101,
		IsTarget:          true,
		TargetSource:      "curriculum",
		TargetSourceRefID: "lesson_1",
		TargetPriority:    0.9,
		Status:            "new",
		EaseFactor:        2.5,
	}

	if _, err := repo.Upsert(context.Background(), state); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	states, err := repo.ListByUser(context.Background(), userID, model.UserUnitStateFilter{})
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("states len = %d, want 1", len(states))
	}

	if err := repo.DeleteByUser(context.Background(), userID); err != nil {
		t.Fatalf("DeleteByUser() error = %v", err)
	}

	states, err = repo.ListByUser(context.Background(), userID, model.UserUnitStateFilter{})
	if err != nil {
		t.Fatalf("ListByUser() after delete error = %v", err)
	}
	if len(states) != 0 {
		t.Fatalf("states len after delete = %d, want 0", len(states))
	}
}

func TestTargetStateCommandRepositoryEnsureAndSetInactive(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)

	targetRepo := persistrepo.NewTargetStateCommandRepository(db.Pool)
	stateRepo := persistrepo.NewUserUnitStateRepository(db.Pool)

	if err := targetRepo.EnsureTargetUnits(context.Background(), userID, []model.TargetUnitSpec{
		{
			CoarseUnitID:      101,
			TargetSource:      "curriculum",
			TargetSourceRefID: "lesson_1",
			TargetPriority:    0.9,
		},
	}); err != nil {
		t.Fatalf("EnsureTargetUnits() error = %v", err)
	}

	states, err := stateRepo.ListByUser(context.Background(), userID, model.UserUnitStateFilter{})
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(states) != 1 || !states[0].IsTarget {
		t.Fatalf("unexpected states after ensure: %+v", states)
	}

	if err := targetRepo.SetTargetInactive(context.Background(), userID, 101); err != nil {
		t.Fatalf("SetTargetInactive() error = %v", err)
	}

	states, err = stateRepo.ListByUser(context.Background(), userID, model.UserUnitStateFilter{})
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if states[0].IsTarget {
		t.Fatalf("is_target = true, want false")
	}
}
