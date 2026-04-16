package selector_test

import (
	"testing"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/domain/policy"
	recommendationselector "learning-video-recommendation-system/internal/recommendation/domain/selector"
)

func TestDefaultVideoSelectorNormalModePreservesCoreCoverage(t *testing.T) {
	selector := recommendationselector.NewDefaultVideoSelector()

	selected, err := selector.Select(model.RecommendationContext{
		Request: model.RecommendationRequest{TargetVideoCount: 4},
	}, []model.VideoCandidate{
		scoredVideo("video-hard-a", string(policy.BucketHardReview), 0.92, 101, []string{string(policy.LaneExactCore)}),
		scoredVideo("video-hard-b", string(policy.BucketNewNow), 0.87, 201, []string{string(policy.LaneBundle)}),
		scoredVideo("video-future-a", string(policy.BucketNearFuture), 0.95, 401, []string{string(policy.LaneSoftFuture)}),
		scoredVideo("video-future-b", string(policy.BucketNearFuture), 0.91, 402, []string{string(policy.LaneSoftFuture)}),
		scoredVideo("video-soft", string(policy.BucketSoftReview), 0.85, 301, []string{string(policy.LaneSoftFuture)}),
	}, normalDemand())
	if err != nil {
		t.Fatalf("select: %v", err)
	}

	if len(selected) != 4 {
		t.Fatalf("expected 4 selected videos, got %#v", selected)
	}
	coreDominant := 0
	futureDominant := 0
	for _, video := range selected {
		if video.DominantBucket == string(policy.BucketHardReview) || video.DominantBucket == string(policy.BucketNewNow) {
			coreDominant++
		}
		if video.DominantBucket == string(policy.BucketNearFuture) {
			futureDominant++
		}
	}
	if coreDominant < 2 {
		t.Fatalf("expected selector to preserve core dominant minimum, got %#v", selected)
	}
	if futureDominant > 1 {
		t.Fatalf("expected future dominant max 1 in normal mode, got %#v", selected)
	}
}

func TestDefaultVideoSelectorLowSupplyAllowsFutureLikeButLimitsFallback(t *testing.T) {
	demand := normalDemand()
	demand.Flags.HardReviewLowSupply = true
	demand.MixQuota = model.MixQuota{
		CoreDominantMin:   2,
		FutureDominantMax: 2,
		FutureLikeMax:     2,
		FallbackMax:       1,
		SameUnitMax:       2,
	}

	selector := recommendationselector.NewDefaultVideoSelector()
	selected, err := selector.Select(model.RecommendationContext{
		Request: model.RecommendationRequest{TargetVideoCount: 4},
	}, []model.VideoCandidate{
		scoredVideo("video-hard", string(policy.BucketHardReview), 0.86, 101, []string{string(policy.LaneExactCore)}),
		scoredVideo("video-soft-a", string(policy.BucketSoftReview), 0.90, 301, []string{string(policy.LaneSoftFuture)}),
		scoredVideo("video-soft-b", string(policy.BucketSoftReview), 0.88, 302, []string{string(policy.LaneBundle)}),
		scoredVideo("video-fallback-a", string(policy.BucketNearFuture), 0.83, 401, []string{string(policy.LaneQualityFallback)}),
		scoredVideo("video-fallback-b", string(policy.BucketNearFuture), 0.82, 402, []string{string(policy.LaneQualityFallback)}),
	}, demand)
	if err != nil {
		t.Fatalf("select: %v", err)
	}

	fallbackCount := 0
	for _, video := range selected {
		if len(video.LaneSources) == 1 && video.LaneSources[0] == string(policy.LaneQualityFallback) {
			fallbackCount++
		}
	}
	if fallbackCount > 1 {
		t.Fatalf("expected fallback max 1 in low supply mode, got %#v", selected)
	}
}

func TestDefaultVideoSelectorUnderfillsWhenConstraintsLeaveNoAdditionalCandidates(t *testing.T) {
	demand := normalDemand()
	selector := recommendationselector.NewDefaultVideoSelector()
	selected, err := selector.Select(model.RecommendationContext{
		Request: model.RecommendationRequest{TargetVideoCount: 4},
	}, []model.VideoCandidate{
		scoredVideo("video-future-only", string(policy.BucketNearFuture), 0.90, 401, []string{string(policy.LaneSoftFuture)}),
	}, demand)
	if err != nil {
		t.Fatalf("select: %v", err)
	}

	if len(selected) != 1 {
		t.Fatalf("expected under-fill in extreme sparse mode, got %#v", selected)
	}
}

func normalDemand() model.DemandBundle {
	return model.DemandBundle{
		HardReview:       []model.DemandUnit{{UnitID: 101}},
		NewNow:           []model.DemandUnit{{UnitID: 201}},
		SoftReview:       []model.DemandUnit{{UnitID: 301}, {UnitID: 302}},
		NearFuture:       []model.DemandUnit{{UnitID: 401}, {UnitID: 402}},
		TargetVideoCount: 4,
		MixQuota: model.MixQuota{
			CoreDominantMin:   2,
			FutureDominantMax: 1,
			FutureLikeMax:     1,
			FallbackMax:       1,
			SameUnitMax:       2,
		},
	}
}

func scoredVideo(videoID string, dominantBucket string, baseScore float64, dominantUnitID int64, laneSources []string) model.VideoCandidate {
	return model.VideoCandidate{
		VideoID:                videoID,
		DominantBucket:         dominantBucket,
		DominantUnitID:         &dominantUnitID,
		BaseScore:              baseScore,
		LaneSources:            append([]string(nil), laneSources...),
		CoveredHardReviewUnits: []int64{101},
		CoveredNewNowUnits:     []int64{201},
		CoveredSoftReviewUnits: []int64{301},
		CoveredNearFutureUnits: []int64{401},
	}
}
