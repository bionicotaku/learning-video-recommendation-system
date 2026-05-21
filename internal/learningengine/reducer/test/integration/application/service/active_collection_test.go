//go:build integration

package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	"learning-video-recommendation-system/internal/learningengine/reducer/application/service"
	persisttx "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/tx"
	"learning-video-recommendation-system/internal/learningengine/reducer/test/fixture"
)

func TestActivateUnitCollectionTargetCreatesProfileAndPreservesLearningState(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-4111-8111-111111111111"
	oldCollectionID := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	newCollectionID := "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	db.SeedCoarseUnit(t, 102)
	db.SeedCoarseUnit(t, 103)
	db.SeedCoarseUnit(t, 104)
	db.SeedUnitCollection(t, oldCollectionID, "ielts-core", "IELTS Core", "active")
	db.SeedUnitCollection(t, newCollectionID, "toefl-core", "TOEFL Core", "active")
	db.SeedUnitCollectionMember(t, oldCollectionID, 101, 1)
	db.SeedUnitCollectionMember(t, oldCollectionID, 102, 2)
	db.SeedUnitCollectionMember(t, newCollectionID, 102, 1)
	db.SeedUnitCollectionMember(t, newCollectionID, 103, 2)
	db.SetCollectionCounts(t, oldCollectionID, 2, 2)
	db.SetCollectionCounts(t, newCollectionID, 2, 2)
	db.SeedUserUnitState(t, userID, 101, "unit_collection", oldCollectionID, true, "new", 0)
	db.SeedUserUnitState(t, userID, 102, "unit_collection", oldCollectionID, true, "reviewing", 88.5)
	db.SeedUserUnitState(t, userID, 104, "manual", "manual-list", true, "learning", 15)

	usecase := service.NewActivateUnitCollectionTargetUsecase(persisttx.NewManager(db.Pool))
	response, err := usecase.Execute(context.Background(), dto.ActivateUnitCollectionTargetRequest{
		UserID:         userID,
		CollectionSlug: "toefl-core",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.CollectionID != newCollectionID || response.CollectionSlug != "toefl-core" || response.TargetCount != 2 {
		t.Fatalf("unexpected response: %+v", response)
	}

	var activeSlug string
	if err := db.Pool.QueryRow(context.Background(), `select active_collection_slug from learning.user_learning_profiles where user_id = $1`, userID).Scan(&activeSlug); err != nil {
		t.Fatalf("read profile: %v", err)
	}
	if activeSlug != "toefl-core" {
		t.Fatalf("active slug = %q, want toefl-core", activeSlug)
	}

	assertState(t, db, userID, 101, false, "unit_collection", oldCollectionID, "new", 0)
	assertState(t, db, userID, 102, true, "unit_collection", newCollectionID, "reviewing", 88.5)
	assertState(t, db, userID, 103, true, "unit_collection", newCollectionID, "new", 0)
	assertState(t, db, userID, 104, true, "manual", "manual-list", "learning", 15)
}

func TestActivateUnitCollectionTargetHandlesEmptyAndMissingCollections(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "22222222-2222-4222-8222-222222222222"
	emptyCollectionID := "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 201)
	db.SeedUnitCollection(t, emptyCollectionID, "empty-book", "Empty Book", "active")
	db.SeedUserUnitState(t, userID, 201, "unit_collection", "old-book", true, "new", 0)

	usecase := service.NewActivateUnitCollectionTargetUsecase(persisttx.NewManager(db.Pool))
	response, err := usecase.Execute(context.Background(), dto.ActivateUnitCollectionTargetRequest{
		UserID:         userID,
		CollectionSlug: "empty-book",
	})
	if err != nil {
		t.Fatalf("Execute(empty) error = %v", err)
	}
	if response.TargetCount != 0 {
		t.Fatalf("TargetCount = %d, want 0", response.TargetCount)
	}
	assertState(t, db, userID, 201, false, "unit_collection", "old-book", "new", 0)

	_, err = usecase.Execute(context.Background(), dto.ActivateUnitCollectionTargetRequest{
		UserID:         userID,
		CollectionSlug: "missing-book",
	})
	if !errors.Is(err, service.ErrUnitCollectionNotFound) {
		t.Fatalf("missing collection err = %v, want ErrUnitCollectionNotFound", err)
	}
}

func assertState(t *testing.T, db *fixture.TestDatabase, userID string, unitID int64, wantTarget bool, wantSource string, wantRef string, wantStatus string, wantProgress float64) {
	t.Helper()

	var gotTarget bool
	var gotSource string
	var gotRef string
	var gotStatus string
	var gotProgress float64
	err := db.Pool.QueryRow(context.Background(), `
		select is_target, coalesce(target_source, ''), coalesce(target_source_ref_id, ''), status, progress_percent::float8
		from learning.user_unit_states
		where user_id = $1 and coarse_unit_id = $2`, userID, unitID).Scan(&gotTarget, &gotSource, &gotRef, &gotStatus, &gotProgress)
	if errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("state for unit %d not found", unitID)
	}
	if err != nil {
		t.Fatalf("read unit %d state: %v", unitID, err)
	}
	if gotTarget != wantTarget || gotSource != wantSource || gotRef != wantRef || gotStatus != wantStatus || gotProgress != wantProgress {
		t.Fatalf("unit %d state = target:%v source:%q ref:%q status:%q progress:%v, want target:%v source:%q ref:%q status:%q progress:%v", unitID, gotTarget, gotSource, gotRef, gotStatus, gotProgress, wantTarget, wantSource, wantRef, wantStatus, wantProgress)
	}
}
