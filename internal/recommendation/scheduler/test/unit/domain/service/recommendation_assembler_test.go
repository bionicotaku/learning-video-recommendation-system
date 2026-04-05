package service_test

import (
	"slices"
	"testing"
	"time"

	appquery "learning-video-recommendation-system/internal/recommendation/scheduler/application/query"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	servicepkg "learning-video-recommendation-system/internal/recommendation/scheduler/domain/service"

	"github.com/google/uuid"
)

func TestRecommendationAssemblerBuildsPredictableBatch(t *testing.T) {
	assembler := servicepkg.NewRecommendationAssembler()
	userID := uuid.New()
	now := time.Date(2026, 4, 7, 14, 0, 0, 0, time.UTC)

	priorityZero := []appquery.ScoredReviewCandidate{
		{
			Candidate: appquery.ReviewCandidate{
				State: model.LearningStateSnapshot{CoarseUnitID: 1, Status: enum.UnitStatusLearning, TargetPriority: 0.9},
				Unit:  model.CoarseUnitRef{CoarseUnitID: 1, Kind: enum.UnitKindWord, Label: "hood"},
			},
			Score:       0.91,
			ReasonCodes: []string{"review_due", "priority_zero_learning_due"},
		},
	}

	scoredReviews := []appquery.ScoredReviewCandidate{
		{
			Candidate: appquery.ReviewCandidate{
				State: model.LearningStateSnapshot{CoarseUnitID: 2, Status: enum.UnitStatusReviewing, TargetPriority: 0.8},
				Unit:  model.CoarseUnitRef{CoarseUnitID: 2, Kind: enum.UnitKindPhrase, Label: "14 July"},
			},
			Score:       0.8,
			ReasonCodes: []string{"review_due", "overdue"},
		},
		{
			Candidate: appquery.ReviewCandidate{
				State: model.LearningStateSnapshot{CoarseUnitID: 1, Status: enum.UnitStatusLearning, TargetPriority: 0.9},
				Unit:  model.CoarseUnitRef{CoarseUnitID: 1, Kind: enum.UnitKindWord, Label: "hood"},
			},
			Score:       0.7,
			ReasonCodes: []string{"review_due"},
		},
	}

	scoredNews := []appquery.ScoredNewCandidate{
		{
			Candidate: appquery.NewCandidate{
				State: model.LearningStateSnapshot{CoarseUnitID: 3, Status: enum.UnitStatusNew, TargetPriority: 0.7},
				Unit:  model.CoarseUnitRef{CoarseUnitID: 3, Kind: enum.UnitKindPhrase, Label: "1 Kings"},
			},
			Score:       0.6,
			ReasonCodes: []string{"new_candidate", "fresh_candidate"},
		},
		{
			Candidate: appquery.NewCandidate{
				State: model.LearningStateSnapshot{CoarseUnitID: 4, Status: enum.UnitStatusNew, TargetPriority: 0.5},
				Unit:  model.CoarseUnitRef{CoarseUnitID: 4, Kind: enum.UnitKindPhrase, Label: ".45 calibre"},
			},
			Score:       0.4,
			ReasonCodes: []string{"new_candidate"},
		},
	}

	batch := assembler.Assemble(userID, now, priorityZero, scoredReviews, scoredNews, servicepkg.QuotaAllocation{
		ReviewQuota:       2,
		NewQuota:          1,
		BacklogProtection: true,
	})

	if batch.RunID == uuid.Nil {
		t.Fatal("RunID = nil, want generated UUID")
	}
	if batch.UserID != userID {
		t.Fatalf("UserID = %v, want %v", batch.UserID, userID)
	}
	if batch.SessionLimit != 3 {
		t.Fatalf("SessionLimit = %d, want 3", batch.SessionLimit)
	}
	if batch.ReviewQuota != 2 || batch.NewQuota != 1 {
		t.Fatalf("quotas = (%d,%d), want (2,1)", batch.ReviewQuota, batch.NewQuota)
	}
	if !batch.BacklogProtection {
		t.Fatal("BacklogProtection = false, want true")
	}
	if len(batch.Items) != 3 {
		t.Fatalf("len(Items) = %d, want 3", len(batch.Items))
	}

	wantOrder := []int64{1, 2, 3}
	for i, want := range wantOrder {
		if batch.Items[i].CoarseUnitID != want {
			t.Fatalf("Items[%d].CoarseUnitID = %d, want %d", i, batch.Items[i].CoarseUnitID, want)
		}
		if batch.Items[i].Rank != i+1 {
			t.Fatalf("Items[%d].Rank = %d, want %d", i, batch.Items[i].Rank, i+1)
		}
	}
	if batch.Items[0].RecommendType != enum.RecommendTypeReview || batch.Items[1].RecommendType != enum.RecommendTypeReview || batch.Items[2].RecommendType != enum.RecommendTypeNew {
		t.Fatalf("RecommendTypes = %v, want review, review, new", []enum.RecommendType{batch.Items[0].RecommendType, batch.Items[1].RecommendType, batch.Items[2].RecommendType})
	}
	if !slices.Contains(batch.Items[0].ReasonCodes, "priority_zero_learning_due") {
		t.Fatalf("Items[0].ReasonCodes = %v, want priority_zero_learning_due", batch.Items[0].ReasonCodes)
	}
	if !slices.Contains(batch.Items[2].ReasonCodes, "fresh_candidate") {
		t.Fatalf("Items[2].ReasonCodes = %v, want fresh_candidate", batch.Items[2].ReasonCodes)
	}
}

func TestRecommendationAssemblerLetsNewFillUnusedReviewCapacity(t *testing.T) {
	assembler := servicepkg.NewRecommendationAssembler()
	userID := uuid.New()
	now := time.Date(2026, 4, 7, 14, 0, 0, 0, time.UTC)

	batch := assembler.Assemble(userID, now, nil, []appquery.ScoredReviewCandidate{
		{
			Candidate: appquery.ReviewCandidate{
				State: model.LearningStateSnapshot{CoarseUnitID: 10, Status: enum.UnitStatusReviewing},
				Unit:  model.CoarseUnitRef{CoarseUnitID: 10, Kind: enum.UnitKindWord, Label: "review-only"},
			},
			Score:       0.8,
			ReasonCodes: []string{"review_due"},
		},
	}, []appquery.ScoredNewCandidate{
		{
			Candidate: appquery.NewCandidate{
				State: model.LearningStateSnapshot{CoarseUnitID: 11, Status: enum.UnitStatusNew},
				Unit:  model.CoarseUnitRef{CoarseUnitID: 11, Kind: enum.UnitKindPhrase, Label: "new-1"},
			},
			Score:       0.7,
			ReasonCodes: []string{"new_candidate"},
		},
		{
			Candidate: appquery.NewCandidate{
				State: model.LearningStateSnapshot{CoarseUnitID: 12, Status: enum.UnitStatusNew},
				Unit:  model.CoarseUnitRef{CoarseUnitID: 12, Kind: enum.UnitKindPhrase, Label: "new-2"},
			},
			Score:       0.6,
			ReasonCodes: []string{"new_candidate"},
		},
	}, servicepkg.QuotaAllocation{
		ReviewQuota: 2,
		NewQuota:    1,
	})

	if len(batch.Items) != 3 {
		t.Fatalf("len(Items) = %d, want 3", len(batch.Items))
	}
	wantOrder := []int64{10, 11, 12}
	for i, want := range wantOrder {
		if batch.Items[i].CoarseUnitID != want {
			t.Fatalf("Items[%d].CoarseUnitID = %d, want %d", i, batch.Items[i].CoarseUnitID, want)
		}
	}
}
