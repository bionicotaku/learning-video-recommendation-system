package service

import (
	"slices"
	"testing"
	"time"

	appquery "learning-video-recommendation-system/internal/recommendation/application/query"
	"learning-video-recommendation-system/internal/recommendation/domain/enum"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

func TestReviewScorerScoreDirectionAndReasons(t *testing.T) {
	scorer := NewReviewScorer()
	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	oldRecommended := now.Add(-7 * time.Hour)
	dueLongAgo := now.Add(-48 * time.Hour)
	lastQuality := 2

	high := scorer.Score(appquery.ReviewCandidate{
		State: model.UserUnitState{
			CoarseUnitID:     1,
			Status:           enum.UnitStatusReviewing,
			TargetPriority:   0.9,
			MasteryScore:     0.1,
			ConsecutiveWrong: 2,
			LastQuality:      &lastQuality,
			NextReviewAt:     &dueLongAgo,
		},
		Serving: model.UserUnitServingState{
			LastRecommendedAt: &oldRecommended,
		},
	}, now)

	low := scorer.Score(appquery.ReviewCandidate{
		State: model.UserUnitState{
			CoarseUnitID:   2,
			Status:         enum.UnitStatusReviewing,
			TargetPriority: 0.2,
			MasteryScore:   0.9,
			NextReviewAt:   ptrTime(now.Add(-1 * time.Hour)),
		},
	}, now)

	if high.Score <= low.Score {
		t.Fatalf("high.Score = %v, low.Score = %v, want high > low", high.Score, low.Score)
	}
	for _, code := range []string{"review_due", "overdue", "weak_memory", "recent_failure", "not_recently_recommended"} {
		if !slices.Contains(high.ReasonCodes, code) {
			t.Fatalf("high.ReasonCodes = %v, want %q", high.ReasonCodes, code)
		}
	}
}

func TestNewScorerScoreDirectionAndReasons(t *testing.T) {
	scorer := NewNewScorer()
	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	oldRecommended := now.Add(-25 * time.Hour)
	recentRecommended := now.Add(-1 * time.Hour)

	fresh := scorer.Score(appquery.NewCandidate{
		State: model.UserUnitState{
			CoarseUnitID:     1,
			TargetPriority:   0.9,
			SeenCount:        0,
			StrongEventCount: 0,
		},
		Serving: model.UserUnitServingState{
			LastRecommendedAt: &oldRecommended,
		},
	}, now)

	stale := scorer.Score(appquery.NewCandidate{
		State: model.UserUnitState{
			CoarseUnitID:     2,
			TargetPriority:   0.6,
			SeenCount:        2,
			StrongEventCount: 0,
		},
		Serving: model.UserUnitServingState{
			LastRecommendedAt: &recentRecommended,
		},
	}, now)

	if fresh.Score <= stale.Score {
		t.Fatalf("fresh.Score = %v, stale.Score = %v, want fresh > stale", fresh.Score, stale.Score)
	}
	for _, code := range []string{"new_candidate", "fresh_candidate", "not_recently_recommended"} {
		if !slices.Contains(fresh.ReasonCodes, code) {
			t.Fatalf("fresh.ReasonCodes = %v, want %q", fresh.ReasonCodes, code)
		}
	}
	if !slices.Contains(stale.ReasonCodes, "recommended_unconsumed") {
		t.Fatalf("stale.ReasonCodes = %v, want recommended_unconsumed", stale.ReasonCodes)
	}
}

func TestPriorityZeroExtractorPrioritizesLearningDueAndRecentFailure(t *testing.T) {
	extractor := NewPriorityZeroExtractor()
	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	badQuality := 2

	items := []appquery.ScoredReviewCandidate{
		{
			Candidate:   appquery.ReviewCandidate{State: model.UserUnitState{CoarseUnitID: 3, Status: enum.UnitStatusReviewing}},
			Score:       0.6,
			ReasonCodes: []string{"review_due"},
		},
		{
			Candidate: appquery.ReviewCandidate{
				State: model.UserUnitState{CoarseUnitID: 1, Status: enum.UnitStatusLearning, NextReviewAt: ptrTime(now.Add(-1 * time.Hour))},
			},
			Score:       0.5,
			ReasonCodes: []string{"review_due"},
		},
		{
			Candidate: appquery.ReviewCandidate{
				State: model.UserUnitState{CoarseUnitID: 2, Status: enum.UnitStatusReviewing, LastQuality: &badQuality},
			},
			Score:       0.7,
			ReasonCodes: []string{"review_due", "recent_failure"},
		},
	}

	got := extractor.Extract(items)
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].Candidate.State.CoarseUnitID != 1 {
		t.Fatalf("got[0].CoarseUnitID = %d, want 1", got[0].Candidate.State.CoarseUnitID)
	}
	if got[1].Candidate.State.CoarseUnitID != 2 {
		t.Fatalf("got[1].CoarseUnitID = %d, want 2", got[1].Candidate.State.CoarseUnitID)
	}
	if !slices.Contains(got[0].ReasonCodes, "priority_zero_learning_due") {
		t.Fatalf("got[0].ReasonCodes = %v, want priority_zero_learning_due", got[0].ReasonCodes)
	}
	if !slices.Contains(got[1].ReasonCodes, "priority_zero_recent_failure") {
		t.Fatalf("got[1].ReasonCodes = %v, want priority_zero_recent_failure", got[1].ReasonCodes)
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
