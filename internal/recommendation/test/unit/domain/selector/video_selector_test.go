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
		scoredVideo("video-hard-a", model.LearningRoleHardReview, 0.92, 101, []string{string(policy.LaneExactCore)}),
		scoredVideo("video-hard-b", model.LearningRoleNewNow, 0.87, 201, []string{string(policy.LaneBundle)}),
		scoredVideo("video-future-a", model.LearningRoleNearFuture, 0.95, 401, []string{string(policy.LaneSoftFuture)}),
		scoredVideo("video-future-b", model.LearningRoleNearFuture, 0.91, 402, []string{string(policy.LaneSoftFuture)}),
		scoredVideo("video-soft", model.LearningRoleSoftReview, 0.85, 301, []string{string(policy.LaneSoftFuture)}),
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
		if video.DominantRole == model.LearningRoleHardReview || video.DominantRole == model.LearningRoleNewNow {
			coreDominant++
		}
		if video.DominantRole == model.LearningRoleNearFuture {
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
		scoredVideo("video-hard", model.LearningRoleHardReview, 0.86, 101, []string{string(policy.LaneExactCore)}),
		scoredVideo("video-soft-a", model.LearningRoleSoftReview, 0.90, 301, []string{string(policy.LaneSoftFuture)}),
		scoredVideo("video-soft-b", model.LearningRoleSoftReview, 0.88, 302, []string{string(policy.LaneBundle)}),
		scoredVideo("video-fallback-a", model.LearningRoleNearFuture, 0.83, 401, []string{string(policy.LaneQualityFallback)}),
		scoredVideo("video-fallback-b", model.LearningRoleNearFuture, 0.82, 402, []string{string(policy.LaneQualityFallback)}),
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

func TestDefaultVideoSelectorLimitsRepeatedPrimaryLearningUnitsAcrossDominantUnits(t *testing.T) {
	demand := normalDemand()
	demand.TargetVideoCount = 3
	demand.MixQuota = model.MixQuota{
		CoreDominantMin:   2,
		FutureDominantMax: 1,
		FutureLikeMax:     1,
		FallbackMax:       1,
		SameUnitMax:       2,
	}

	selector := recommendationselector.NewDefaultVideoSelector()
	selected, err := selector.Select(model.RecommendationContext{
		Request: model.RecommendationRequest{TargetVideoCount: 3},
	}, []model.VideoCandidate{
		scoredVideoWithUnits("video-a", model.LearningRoleHardReview, 0.93, 101, []string{string(policy.LaneExactCore)},
			learningUnit(101, model.LearningRoleHardReview, true),
		),
		scoredVideoWithUnits("video-b", model.LearningRoleHardReview, 0.92, 102, []string{string(policy.LaneExactCore)},
			learningUnit(102, model.LearningRoleHardReview, true),
			learningUnit(101, model.LearningRoleHardReview, true),
		),
		scoredVideoWithUnits("video-c", model.LearningRoleHardReview, 0.91, 103, []string{string(policy.LaneExactCore)},
			learningUnit(103, model.LearningRoleHardReview, true),
			learningUnit(101, model.LearningRoleHardReview, true),
		),
	}, demand)
	if err != nil {
		t.Fatalf("select: %v", err)
	}

	if len(selected) != 2 {
		t.Fatalf("expected same primary unit cap to under-fill at 2 videos, got %#v", selected)
	}
	if containsVideo(selected, "video-c") {
		t.Fatalf("expected video-c to be blocked by repeated primary unit 101, got %#v", selected)
	}
}

func TestDefaultVideoSelectorAllowsRepeatedNonPrimarySupportUnits(t *testing.T) {
	demand := normalDemand()
	demand.TargetVideoCount = 2
	demand.MixQuota = model.MixQuota{
		CoreDominantMin:   1,
		FutureDominantMax: 1,
		FutureLikeMax:     1,
		FallbackMax:       1,
		SameUnitMax:       1,
	}

	selector := recommendationselector.NewDefaultVideoSelector()
	selected, err := selector.Select(model.RecommendationContext{
		Request: model.RecommendationRequest{TargetVideoCount: 2},
	}, []model.VideoCandidate{
		scoredVideoWithUnits("video-a", model.LearningRoleHardReview, 0.93, 101, []string{string(policy.LaneExactCore)},
			learningUnit(101, model.LearningRoleHardReview, true),
			learningUnit(301, model.LearningRoleSoftReview, false),
		),
		scoredVideoWithUnits("video-b", model.LearningRoleHardReview, 0.92, 102, []string{string(policy.LaneExactCore)},
			learningUnit(102, model.LearningRoleHardReview, true),
			learningUnit(301, model.LearningRoleSoftReview, false),
		),
	}, demand)
	if err != nil {
		t.Fatalf("select: %v", err)
	}

	if len(selected) != 2 {
		t.Fatalf("expected repeated non-primary support unit to stay selectable, got %#v", selected)
	}
}

func TestDefaultVideoSelectorUnderfillsWhenConstraintsLeaveNoAdditionalCandidates(t *testing.T) {
	demand := normalDemand()
	selector := recommendationselector.NewDefaultVideoSelector()
	selected, err := selector.Select(model.RecommendationContext{
		Request: model.RecommendationRequest{TargetVideoCount: 4},
	}, []model.VideoCandidate{
		scoredVideo("video-future-only", model.LearningRoleNearFuture, 0.90, 401, []string{string(policy.LaneSoftFuture)}),
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

func scoredVideo(videoID string, dominantRole model.LearningRole, baseScore float64, dominantUnitID int64, laneSources []string) model.VideoCandidate {
	return scoredVideoWithUnits(videoID, dominantRole, baseScore, dominantUnitID, laneSources,
		learningUnit(dominantUnitID, dominantRole, model.IsCoreLearningRole(dominantRole)),
	)
}

func scoredVideoWithUnits(videoID string, dominantRole model.LearningRole, baseScore float64, dominantUnitID int64, laneSources []string, units ...model.ExpectedLearningUnit) model.VideoCandidate {
	return model.VideoCandidate{
		VideoID:        videoID,
		DominantRole:   dominantRole,
		DominantUnitID: &dominantUnitID,
		BaseScore:      baseScore,
		LaneSources:    append([]string(nil), laneSources...),
		LearningUnits:  append([]model.ExpectedLearningUnit(nil), units...),
	}
}

func learningUnit(unitID int64, role model.LearningRole, primary bool) model.ExpectedLearningUnit {
	return model.ExpectedLearningUnit{
		CoarseUnitID: unitID,
		Role:         role,
		IsPrimary:    primary,
	}
}

func containsVideo(videos []model.VideoCandidate, videoID string) bool {
	for _, video := range videos {
		if video.VideoID == videoID {
			return true
		}
	}
	return false
}
