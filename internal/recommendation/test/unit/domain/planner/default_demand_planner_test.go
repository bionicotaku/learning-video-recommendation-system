package planner_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
	recommendationplanner "learning-video-recommendation-system/internal/recommendation/domain/planner"
)

func TestDefaultDemandPlannerRespectsBucketPrecedence(t *testing.T) {
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	lastQuality := int16(2)

	planner := recommendationplanner.NewDefaultDemandPlanner()
	bundle, err := planner.Plan(model.RecommendationContext{
		Now: now,
		Request: model.RecommendationRequest{
			TargetVideoCount:     8,
			PreferredDurationSec: [2]int{45, 180},
		},
		ActiveUnitStates: []model.LearningStateSnapshot{
			{
				CoarseUnitID:   101,
				Status:         "new",
				TargetPriority: 0.9,
				LastQuality:    &lastQuality,
			},
		},
		UnitInventory: []model.UnitVideoInventory{
			{CoarseUnitID: 101, SupplyGrade: "ok"},
		},
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	if len(bundle.HardReview) != 1 {
		t.Fatalf("expected 1 hard review unit, got %#v", bundle)
	}
	if len(bundle.NewNow) != 0 {
		t.Fatalf("expected no new_now units, got %#v", bundle.NewNow)
	}
}

func TestDefaultDemandPlannerTreatsUnsuppliedNewUnitsAsNearFuture(t *testing.T) {
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	planner := recommendationplanner.NewDefaultDemandPlanner()

	bundle, err := planner.Plan(model.RecommendationContext{
		Now: now,
		Request: model.RecommendationRequest{
			TargetVideoCount:     8,
			PreferredDurationSec: [2]int{45, 180},
		},
		ActiveUnitStates: []model.LearningStateSnapshot{
			{
				CoarseUnitID:   202,
				Status:         "new",
				TargetPriority: 0.7,
			},
		},
		UnitInventory: []model.UnitVideoInventory{
			{CoarseUnitID: 202, SupplyGrade: "none"},
		},
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	if len(bundle.NewNow) != 0 {
		t.Fatalf("expected no new_now units, got %#v", bundle.NewNow)
	}
	if len(bundle.NearFuture) != 1 {
		t.Fatalf("expected 1 near_future unit, got %#v", bundle.NearFuture)
	}
}

func TestDefaultDemandPlannerRaisesBundleBudgetWhenHardReviewSupplyIsWeak(t *testing.T) {
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	dueAt := now.Add(-time.Hour)
	planner := recommendationplanner.NewDefaultDemandPlanner()

	bundle, err := planner.Plan(model.RecommendationContext{
		Now: now,
		Request: model.RecommendationRequest{
			TargetVideoCount:     8,
			PreferredDurationSec: [2]int{45, 180},
		},
		ActiveUnitStates: []model.LearningStateSnapshot{
			{
				CoarseUnitID:   303,
				Status:         "reviewing",
				TargetPriority: 1.0,
				NextReviewAt:   &dueAt,
			},
		},
		UnitInventory: []model.UnitVideoInventory{
			{CoarseUnitID: 303, SupplyGrade: "weak"},
		},
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	if !bundle.Flags.HardReviewLowSupply {
		t.Fatal("expected HardReviewLowSupply flag")
	}
	if bundle.LaneBudget.Bundle != 0.35 || bundle.LaneBudget.SoftFuture != 0.20 {
		t.Fatalf("unexpected lane budget: %#v", bundle.LaneBudget)
	}
}

func TestDefaultDemandPlannerSeparatesSoftReviewAndNearFuture(t *testing.T) {
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	softDue := now.Add(48 * time.Hour)
	futureDue := now.Add(10 * 24 * time.Hour)
	planner := recommendationplanner.NewDefaultDemandPlanner()

	bundle, err := planner.Plan(model.RecommendationContext{
		Now: now,
		Request: model.RecommendationRequest{
			TargetVideoCount:     8,
			PreferredDurationSec: [2]int{45, 180},
		},
		ActiveUnitStates: []model.LearningStateSnapshot{
			{CoarseUnitID: 401, Status: "reviewing", TargetPriority: 0.8, NextReviewAt: &softDue},
			{CoarseUnitID: 402, Status: "mastered", TargetPriority: 0.6, NextReviewAt: &futureDue, MasteryScore: 0.9},
		},
		UnitInventory: []model.UnitVideoInventory{
			{CoarseUnitID: 401, SupplyGrade: "strong"},
			{CoarseUnitID: 402, SupplyGrade: "strong"},
		},
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	if len(bundle.SoftReview) != 1 || bundle.SoftReview[0].UnitID != 401 {
		t.Fatalf("expected soft review unit 401, got %#v", bundle.SoftReview)
	}
	if len(bundle.NearFuture) != 1 || bundle.NearFuture[0].UnitID != 402 {
		t.Fatalf("expected near future unit 402, got %#v", bundle.NearFuture)
	}
}

func TestDefaultDemandPlannerGoldenReviewHeavy(t *testing.T) {
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	dueAt := now.Add(-2 * time.Hour)
	softAt := now.Add(48 * time.Hour)
	futureAt := now.Add(7 * 24 * time.Hour)

	planner := recommendationplanner.NewDefaultDemandPlanner()
	bundle, err := planner.Plan(model.RecommendationContext{
		Now: now,
		Request: model.RecommendationRequest{
			TargetVideoCount:     8,
			PreferredDurationSec: [2]int{45, 180},
		},
		ActiveUnitStates: []model.LearningStateSnapshot{
			{CoarseUnitID: 101, Status: "reviewing", TargetPriority: 0.9, NextReviewAt: &dueAt},
			{CoarseUnitID: 201, Status: "new", TargetPriority: 0.8},
			{CoarseUnitID: 301, Status: "learning", TargetPriority: 0.7, NextReviewAt: &softAt, MasteryScore: 0.5},
			{CoarseUnitID: 401, Status: "mastered", TargetPriority: 0.6, NextReviewAt: &futureAt, MasteryScore: 0.9},
		},
		UnitInventory: []model.UnitVideoInventory{
			{CoarseUnitID: 101, SupplyGrade: "weak"},
			{CoarseUnitID: 201, SupplyGrade: "ok"},
			{CoarseUnitID: 301, SupplyGrade: "strong"},
			{CoarseUnitID: 401, SupplyGrade: "strong"},
		},
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	actual, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file")
	}
	goldenPath := filepath.Join(filepath.Dir(currentFile), "../../../golden/planner_review_heavy.json")
	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if !bytes.Equal(bytes.TrimSpace(actual), bytes.TrimSpace(expected)) {
		t.Fatalf("planner golden mismatch\nactual:\n%s\nexpected:\n%s", actual, expected)
	}
}
