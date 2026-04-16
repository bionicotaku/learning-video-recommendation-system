package ranking_test

import (
	"testing"
	"time"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/domain/policy"
	recommendationranking "learning-video-recommendation-system/internal/recommendation/domain/ranking"
)

func TestDefaultVideoRankerAppliesFormulaAndPenalties(t *testing.T) {
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	recentServedAt := now.Add(-6 * time.Hour)
	recentWatchedAt := now.Add(-3 * time.Hour)

	ranker := recommendationranking.NewDefaultVideoRanker()
	ranked, err := ranker.Rank(model.RecommendationContext{
		Now: now,
		Request: model.RecommendationRequest{
			TargetVideoCount:     4,
			PreferredDurationSec: [2]int{45, 180},
		},
		VideoServingStates: []model.UserVideoServingState{
			{VideoID: "video-penalized", LastServedAt: &recentServedAt, ServedCount: 3},
		},
		VideoUserStates: []model.VideoUserState{
			{VideoID: "video-penalized", LastWatchedAt: &recentWatchedAt, WatchCount: 4, CompletedCount: 2, MaxWatchRatio: 0.95},
		},
	}, []model.VideoCandidate{
		videoCandidate("video-fresh", string(policy.BucketHardReview), 0.8, 0.7, 0.5, 0.7, 0.2, 120_000, []int64{101}, nil),
		videoCandidate("video-penalized", string(policy.BucketHardReview), 0.8, 0.7, 0.5, 0.7, 0.2, 120_000, []int64{101}, int64Ptr(101)),
	}, recommendationDemand())
	if err != nil {
		t.Fatalf("rank: %v", err)
	}

	if ranked[0].VideoID != "video-fresh" {
		t.Fatalf("expected fresh video to outrank penalized one, got %#v", ranked)
	}
	if ranked[1].RecentServedPenalty <= 0 {
		t.Fatalf("expected recent served penalty, got %#v", ranked[1])
	}
	if ranked[1].RecentWatchedPenalty <= 0 {
		t.Fatalf("expected recent watched penalty, got %#v", ranked[1])
	}
	if ranked[0].BaseScore == ranked[1].BaseScore {
		t.Fatalf("expected ranking score difference, got %#v", ranked)
	}
}

func TestDefaultVideoRankerDoesNotSubtractRecentWatchedPenaltyFromBaseScore(t *testing.T) {
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	recentWatchedAt := now.Add(-3 * time.Hour)

	ranker := recommendationranking.NewDefaultVideoRanker()
	ranked, err := ranker.Rank(model.RecommendationContext{
		Now: now,
		Request: model.RecommendationRequest{
			TargetVideoCount:     1,
			PreferredDurationSec: [2]int{45, 180},
		},
		VideoUserStates: []model.VideoUserState{
			{VideoID: "video-1", LastWatchedAt: &recentWatchedAt, WatchCount: 4, CompletedCount: 2, MaxWatchRatio: 0.95},
		},
	}, []model.VideoCandidate{
		videoCandidate("video-1", string(policy.BucketHardReview), 0.8, 0.7, 0.5, 0.7, 0.2, 120_000, []int64{101}, int64Ptr(101)),
	}, recommendationDemand())
	if err != nil {
		t.Fatalf("rank: %v", err)
	}

	got := ranked[0]
	demandCoverage := 0.50*got.HardReviewCover + 0.20*got.NewNowCover + 0.20*got.SoftReviewCover + 0.10*got.NearFutureCover
	want := round4ForTest(
		0.40*demandCoverage +
			0.18*got.CoverageStrengthScore +
			0.15*got.BundleValueScore +
			0.15*got.EducationalFitScore +
			0.05*got.FutureValueScore +
			0.05*got.FreshnessScore -
			0.03*got.RecentServedPenalty -
			0.02*got.OverloadPenalty,
	)
	if got.BaseScore != want {
		t.Fatalf("expected base score without direct recent watched subtraction, got=%0.4f want=%0.4f candidate=%+v", got.BaseScore, want, got)
	}
}

func TestDefaultVideoRankerAddsOverloadPenaltyForOverstuffedLongVideo(t *testing.T) {
	ranker := recommendationranking.NewDefaultVideoRanker()

	ranked, err := ranker.Rank(model.RecommendationContext{
		Request: model.RecommendationRequest{
			TargetVideoCount:     4,
			PreferredDurationSec: [2]int{45, 180},
		},
	}, []model.VideoCandidate{
		videoCandidate("video-compact", string(policy.BucketHardReview), 0.7, 0.6, 0.4, 0.6, 0.2, 140_000, []int64{101, 201}, int64Ptr(101)),
		videoCandidate("video-overloaded", string(policy.BucketHardReview), 0.7, 0.6, 0.4, 0.6, 0.2, 420_000, []int64{101, 201, 301, 401}, int64Ptr(101)),
	}, recommendationDemand())
	if err != nil {
		t.Fatalf("rank: %v", err)
	}

	if ranked[1].VideoID != "video-overloaded" {
		t.Fatalf("expected overloaded video to rank lower, got %#v", ranked)
	}
	if ranked[1].OverloadPenalty <= 0 {
		t.Fatalf("expected overload penalty, got %#v", ranked[1])
	}
}

func recommendationDemand() model.DemandBundle {
	return model.DemandBundle{
		HardReview:       []model.DemandUnit{{UnitID: 101}, {UnitID: 102}},
		NewNow:           []model.DemandUnit{{UnitID: 201}},
		SoftReview:       []model.DemandUnit{{UnitID: 301}},
		NearFuture:       []model.DemandUnit{{UnitID: 401}},
		TargetVideoCount: 4,
	}
}

func videoCandidate(videoID string, dominantBucket string, hardCover float64, coverageStrength float64, bundleValue float64, fit float64, future float64, bestEndMs int32, coveredUnits []int64, dominantUnitID *int64) model.VideoCandidate {
	start := int32(1000)
	return model.VideoCandidate{
		VideoID:                videoID,
		DominantBucket:         dominantBucket,
		DominantUnitID:         dominantUnitID,
		CoveredHardReviewUnits: append([]int64(nil), coveredUnits...),
		HardReviewCover:        hardCover,
		CoverageStrengthScore:  coverageStrength,
		BundleValueScore:       bundleValue,
		EducationalFitScore:    fit,
		FutureValueScore:       future,
		BestEvidenceStartMs:    &start,
		BestEvidenceEndMs:      &bestEndMs,
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}

func int32Ptr(value int32) *int32 {
	return &value
}

func round4ForTest(value float64) float64 {
	return float64(int(value*10000+0.5)) / 10000
}
