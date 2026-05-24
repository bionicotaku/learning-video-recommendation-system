package planner_test

import (
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
		Now:                  now,
		PreferredDurationSec: [2]int{45, 200},
		Request: model.RecommendationRequest{
			TargetVideoCount: 8,
		},
		ActiveUnitStates: []model.LearningStateSnapshot{
			{
				CoarseUnitID:        101,
				Status:              "new",
				TargetPriority:      0.9,
				LastProgressQuality: &lastQuality,
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
		Now:                  now,
		PreferredDurationSec: [2]int{45, 200},
		Request: model.RecommendationRequest{
			TargetVideoCount: 8,
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
		Now:                  now,
		PreferredDurationSec: [2]int{45, 200},
		Request: model.RecommendationRequest{
			TargetVideoCount: 8,
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
	if bundle.LaneBudget.Bundle <= bundle.LaneBudget.SoftFuture {
		t.Fatalf("expected bundle lane to remain stronger than soft_future under low supply: %#v", bundle.LaneBudget)
	}
	if bundle.LaneBudget.Bundle <= 0 || bundle.LaneBudget.SoftFuture <= 0 {
		t.Fatalf("expected positive expansion lane budgets under low supply: %#v", bundle.LaneBudget)
	}
}

func TestDefaultDemandPlannerSeparatesSoftReviewAndNearFuture(t *testing.T) {
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	softDue := now.Add(48 * time.Hour)
	futureDue := now.Add(10 * 24 * time.Hour)
	planner := recommendationplanner.NewDefaultDemandPlanner()

	bundle, err := planner.Plan(model.RecommendationContext{
		Now:                  now,
		PreferredDurationSec: [2]int{45, 200},
		Request: model.RecommendationRequest{
			TargetVideoCount: 8,
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
