package service_test

import (
	"context"
	"testing"
	"time"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	appservice "learning-video-recommendation-system/internal/recommendation/application/service"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

func TestRecallQueueServiceRebuildsMissingQueueAndSelectsScopedUnits(t *testing.T) {
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	repo := &stubRecallQueueRepository{
		learningVersion: apprepo.LearningStateVersion{
			ActiveTargetUnitCount:      160,
			SourceLearningMaxUpdatedAt: timePtr(now.Add(-time.Hour)),
		},
		projectionUpdatedAt: now.Add(-time.Minute),
		candidates:          recallCandidates(60, 40, 40, 40),
	}

	selection, err := appservice.NewRecallQueueService(repo).SelectScope(context.Background(), "user-1", 8, now)
	if err != nil {
		t.Fatalf("select scope: %v", err)
	}
	scope := selection.PlannerScope
	summary := selection.Summary

	if !summary.QueueRebuilt || repo.rebuildCount != 1 {
		t.Fatalf("expected missing queue to rebuild")
	}
	if len(scope) != 96 {
		t.Fatalf("expected default scope 96, got %d", len(scope))
	}
	if summary.PlannerScopeUnitCountByBucket["hard_review"] != 49 {
		t.Fatalf("expected hard backlog dynamic quota 48, got %#v", summary.PlannerScopeUnitCountByBucket)
	}
	if len(selection.RecallFetchScope) != len(selection.PlannerScope) {
		t.Fatalf("all supplied scope should be fetched, got planner=%d fetch=%d", len(selection.PlannerScope), len(selection.RecallFetchScope))
	}
	if summary.PerUnitRecallLimit != 32 || summary.MaxPossibleRecallRows != 3072 {
		t.Fatalf("unexpected recall row limits: %#v", summary)
	}
}

func TestRecallQueueServiceSkipsFreshRebuild(t *testing.T) {
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	learningUpdated := now.Add(-time.Hour)
	projectionUpdated := now.Add(-time.Minute)
	repo := &stubRecallQueueRepository{
		stateExists: true,
		state: model.RecallQueueState{
			UserID:                     "user-1",
			SourceLearningMaxUpdatedAt: timePtr(learningUpdated),
			SourceProjectionUpdatedAt:  projectionUpdated,
			ActiveTargetUnitCount:      2,
			RebuiltAt:                  now.Add(-time.Minute),
		},
		learningVersion: apprepo.LearningStateVersion{
			ActiveTargetUnitCount:      2,
			SourceLearningMaxUpdatedAt: timePtr(learningUpdated),
		},
		projectionUpdatedAt: projectionUpdated,
		candidates:          recallCandidates(1, 1, 0, 0),
	}

	selection, err := appservice.NewRecallQueueService(repo).SelectScope(context.Background(), "user-1", 8, now)
	if err != nil {
		t.Fatalf("select scope: %v", err)
	}
	summary := selection.Summary
	if summary.QueueRebuilt || repo.rebuildCount != 0 {
		t.Fatalf("expected fresh queue to skip rebuild")
	}
}

func TestRecallQueueServiceRebuildsWhenActiveTargetCountChanges(t *testing.T) {
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	learningUpdated := now.Add(-time.Hour)
	projectionUpdated := now.Add(-time.Minute)
	repo := &stubRecallQueueRepository{
		stateExists: true,
		state: model.RecallQueueState{
			UserID:                     "user-1",
			SourceLearningMaxUpdatedAt: timePtr(learningUpdated),
			SourceProjectionUpdatedAt:  projectionUpdated,
			ActiveTargetUnitCount:      10,
			RebuiltAt:                  now.Add(-time.Minute),
		},
		learningVersion: apprepo.LearningStateVersion{
			ActiveTargetUnitCount:      9,
			SourceLearningMaxUpdatedAt: timePtr(learningUpdated),
		},
		projectionUpdatedAt: projectionUpdated,
		candidates:          recallCandidates(1, 1, 0, 0),
	}

	selection, err := appservice.NewRecallQueueService(repo).SelectScope(context.Background(), "user-1", 8, now)
	if err != nil {
		t.Fatalf("select scope: %v", err)
	}
	summary := selection.Summary
	if !summary.QueueRebuilt || repo.rebuildCount != 1 {
		t.Fatalf("expected active target count change to rebuild")
	}
}

func TestRecallQueueServiceRefillsBucketShortage(t *testing.T) {
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	repo := &stubRecallQueueRepository{
		stateExists:         true,
		state:               freshQueueState(now, 120),
		learningVersion:     apprepo.LearningStateVersion{ActiveTargetUnitCount: 120, SourceLearningMaxUpdatedAt: timePtr(now.Add(-time.Hour))},
		projectionUpdatedAt: now.Add(-time.Minute),
		candidates:          recallCandidates(2, 90, 40, 0),
	}

	selection, err := appservice.NewRecallQueueService(repo).SelectScope(context.Background(), "user-1", 8, now)
	if err != nil {
		t.Fatalf("select scope: %v", err)
	}
	scope := selection.PlannerScope
	summary := selection.Summary
	if len(scope) != 96 {
		t.Fatalf("expected refill to full scope, got %d", len(scope))
	}
	if summary.PlannerScopeUnitCountByBucket["hard_review"] != 2 {
		t.Fatalf("expected only available hard units, got %#v", summary.PlannerScopeUnitCountByBucket)
	}
	if summary.PlannerScopeUnitCountByBucket["new_now"] <= 29 {
		t.Fatalf("expected new_now to receive refill, got %#v", summary.PlannerScopeUnitCountByBucket)
	}
}

func TestRecallQueueServiceCapsNoSupplyUnitsInScope(t *testing.T) {
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	repo := &stubRecallQueueRepository{
		stateExists:         true,
		state:               freshQueueState(now, 140),
		learningVersion:     apprepo.LearningStateVersion{ActiveTargetUnitCount: 140, SourceLearningMaxUpdatedAt: timePtr(now.Add(-time.Hour))},
		projectionUpdatedAt: now.Add(-time.Minute),
		candidates: append(
			recallCandidatesWithSupply("none", 120, 1),
			recallCandidatesWithSupply("ok", 20, 1000)...,
		),
	}

	selection, err := appservice.NewRecallQueueService(repo).SelectScope(context.Background(), "user-1", 8, now)
	if err != nil {
		t.Fatalf("select scope: %v", err)
	}
	scope := selection.PlannerScope
	summary := selection.Summary
	noSupplyCount := 0
	for _, candidate := range scope {
		if candidate.SupplyGrade == "none" {
			noSupplyCount++
		}
	}
	if noSupplyCount > 8 {
		t.Fatalf("expected no-supply units capped at 8, got %d in %#v", noSupplyCount, scope)
	}
	for _, candidate := range selection.RecallFetchScope {
		if candidate.SupplyGrade == "none" {
			t.Fatalf("recall fetch scope contains no-supply candidate: %#v", candidate)
		}
	}
	if summary.NoSupplyScopeUnitCount != noSupplyCount {
		t.Fatalf("no-supply summary count = %d, want %d", summary.NoSupplyScopeUnitCount, noSupplyCount)
	}
	if summary.RecallFetchUnitCount != len(selection.RecallFetchScope) {
		t.Fatalf("recall fetch count = %d, want %d", summary.RecallFetchUnitCount, len(selection.RecallFetchScope))
	}
	if summary.MaxPossibleRecallRows != len(selection.RecallFetchScope)*int(summary.PerUnitRecallLimit) {
		t.Fatalf("max possible recall rows should use fetch units, got %#v", summary)
	}
}

type stubRecallQueueRepository struct {
	stateExists         bool
	state               model.RecallQueueState
	learningVersion     apprepo.LearningStateVersion
	projectionUpdatedAt time.Time
	candidates          []model.RecallQueueCandidate
	rebuildCount        int
}

func (s *stubRecallQueueRepository) GetLearningStateVersion(context.Context, string) (apprepo.LearningStateVersion, error) {
	return s.learningVersion, nil
}

func (s *stubRecallQueueRepository) GetProjectionUpdatedAt(context.Context) (time.Time, error) {
	return s.projectionUpdatedAt, nil
}

func (s *stubRecallQueueRepository) GetQueueState(context.Context, string) (model.RecallQueueState, bool, error) {
	return s.state, s.stateExists, nil
}

func (s *stubRecallQueueRepository) RebuildUserQueue(context.Context, string, time.Time) (model.RecallQueueState, error) {
	s.rebuildCount++
	return model.RecallQueueState{
		UserID:                     "user-1",
		SourceLearningMaxUpdatedAt: s.learningVersion.SourceLearningMaxUpdatedAt,
		SourceProjectionUpdatedAt:  s.projectionUpdatedAt,
		ActiveTargetUnitCount:      s.learningVersion.ActiveTargetUnitCount,
		RebuiltAt:                  time.Now().UTC(),
	}, nil
}

func (s *stubRecallQueueRepository) ListCandidates(context.Context, string, time.Time, int32, int32) ([]model.RecallQueueCandidate, error) {
	return s.candidates, nil
}

func recallCandidates(hardCount int, newCount int, softCount int, futureCount int) []model.RecallQueueCandidate {
	result := make([]model.RecallQueueCandidate, 0, hardCount+newCount+softCount+futureCount)
	nextID := int64(1)
	appendBucket := func(bucket string, count int) {
		for i := 0; i < count; i++ {
			result = append(result, model.RecallQueueCandidate{
				UserID:          "user-1",
				CoarseUnitID:    nextID,
				Status:          "learning",
				SupplyGrade:     "ok",
				Bucket:          bucket,
				DynamicPriority: float64(1000 - nextID),
				StateUpdatedAt:  time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC),
			})
			nextID++
		}
	}
	appendBucket("hard_review", hardCount)
	appendBucket("new_now", newCount)
	appendBucket("soft_review", softCount)
	appendBucket("near_future", futureCount)
	return result
}

func recallCandidatesWithSupply(supplyGrade string, count int, startID int64) []model.RecallQueueCandidate {
	result := make([]model.RecallQueueCandidate, 0, count)
	for i := 0; i < count; i++ {
		result = append(result, model.RecallQueueCandidate{
			UserID:          "user-1",
			CoarseUnitID:    startID + int64(i),
			Status:          "learning",
			SupplyGrade:     supplyGrade,
			Bucket:          "hard_review",
			DynamicPriority: float64(1000 - i),
			StateUpdatedAt:  time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC),
		})
	}
	return result
}

func freshQueueState(now time.Time, count int32) model.RecallQueueState {
	return model.RecallQueueState{
		UserID:                     "user-1",
		SourceLearningMaxUpdatedAt: timePtr(now.Add(-time.Hour)),
		SourceProjectionUpdatedAt:  now.Add(-time.Minute),
		ActiveTargetUnitCount:      count,
		RebuiltAt:                  now.Add(-time.Minute),
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}
