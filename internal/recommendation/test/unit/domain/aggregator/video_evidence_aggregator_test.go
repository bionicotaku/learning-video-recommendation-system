package aggregator_test

import (
	"testing"

	recommendationaggregator "learning-video-recommendation-system/internal/recommendation/domain/aggregator"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/domain/policy"
)

func TestDefaultVideoEvidenceAggregatorDeduplicatesRepeatedUnitEvidence(t *testing.T) {
	aggregator := recommendationaggregator.NewDefaultVideoEvidenceAggregator()

	videos, err := aggregator.Aggregate(recommendationContext(), []model.ResolvedEvidenceWindow{
		resolvedWindow("video-1", 101, string(policy.BucketHardReview), string(policy.LaneExactCore), 0.90, 1000, 1500, 120_000, 0.40),
		resolvedWindow("video-1", 101, string(policy.BucketHardReview), string(policy.LaneExactCore), 0.70, 2000, 2500, 120_000, 0.30),
	}, recommendationDemand())
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}

	if len(videos) != 1 {
		t.Fatalf("expected one video candidate, got %#v", videos)
	}
	if len(videos[0].LearningUnits) != 1 {
		t.Fatalf("expected one expected learning unit, got %#v", videos[0].LearningUnits)
	}
	if videos[0].LearningUnits[0].CoarseUnitID != 101 || videos[0].LearningUnits[0].Role != model.LearningRoleHardReview {
		t.Fatalf("expected hard review learning unit 101, got %#v", videos[0].LearningUnits)
	}
	if videos[0].CoverageStrengthScore >= 0.95 {
		t.Fatalf("expected repeated same-unit evidence to be dampened, got %#v", videos[0].CoverageStrengthScore)
	}
}

func TestDefaultVideoEvidenceAggregatorTracksBucketCoverage(t *testing.T) {
	aggregator := recommendationaggregator.NewDefaultVideoEvidenceAggregator()

	videos, err := aggregator.Aggregate(recommendationContext(), []model.ResolvedEvidenceWindow{
		resolvedWindow("video-2", 101, string(policy.BucketHardReview), string(policy.LaneExactCore), 0.95, 1000, 1400, 120_000, 0.42),
		resolvedWindow("video-2", 201, string(policy.BucketNewNow), string(policy.LaneBundle), 0.80, 2000, 2500, 120_000, 0.31),
		resolvedWindow("video-2", 301, string(policy.BucketSoftReview), string(policy.LaneBundle), 0.75, 3000, 3600, 120_000, 0.28),
		resolvedWindow("video-2", 401, string(policy.BucketNearFuture), string(policy.LaneSoftFuture), 0.60, 4000, 4500, 120_000, 0.18),
	}, recommendationDemand())
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}

	video := videos[0]
	if video.HardReviewCover != 1 {
		t.Fatalf("expected full hard review coverage, got %#v", video.HardReviewCover)
	}
	if video.NewNowCover != 0.5 {
		t.Fatalf("expected half new_now coverage, got %#v", video.NewNowCover)
	}
	if video.SoftReviewCover != 0.5 {
		t.Fatalf("expected half soft_review coverage, got %#v", video.SoftReviewCover)
	}
	if video.NearFutureCover != 0.5 {
		t.Fatalf("expected half near_future coverage, got %#v", video.NearFutureCover)
	}
}

func TestDefaultVideoEvidenceAggregatorChoosesBestEvidenceFromDominantCoreWindow(t *testing.T) {
	aggregator := recommendationaggregator.NewDefaultVideoEvidenceAggregator()

	videos, err := aggregator.Aggregate(recommendationContext(), []model.ResolvedEvidenceWindow{
		resolvedWindow("video-3", 101, string(policy.BucketHardReview), string(policy.LaneExactCore), 0.88, 1000, 1450, 120_000, 0.36),
		resolvedWindow("video-3", 401, string(policy.BucketNearFuture), string(policy.LaneSoftFuture), 0.95, 5000, 5600, 120_000, 0.33),
	}, recommendationDemand())
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}

	video := videos[0]
	if video.DominantRole != model.LearningRoleHardReview {
		t.Fatalf("expected hard_review dominant role, got %#v", video.DominantRole)
	}
	if len(video.LearningUnits) == 0 || video.LearningUnits[0].Evidence == nil || video.LearningUnits[0].Evidence.StartMs == nil || *video.LearningUnits[0].Evidence.StartMs != 1000 {
		t.Fatalf("expected best evidence from core learning unit, got %#v", video.LearningUnits)
	}
}

func TestDefaultVideoEvidenceAggregatorMarksSupportUnitPrimaryWhenNoCoreExists(t *testing.T) {
	aggregator := recommendationaggregator.NewDefaultVideoEvidenceAggregator()

	videos, err := aggregator.Aggregate(recommendationContext(), []model.ResolvedEvidenceWindow{
		resolvedWindow("video-4", 301, string(policy.BucketSoftReview), string(policy.LaneSoftFuture), 0.91, 1000, 1450, 120_000, 0.36),
		resolvedWindow("video-4", 401, string(policy.BucketNearFuture), string(policy.LaneSoftFuture), 0.86, 5000, 5600, 120_000, 0.33),
	}, recommendationDemand())
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}

	if len(videos) != 1 {
		t.Fatalf("expected one video, got %#v", videos)
	}
	primaryCount := 0
	for _, unit := range videos[0].LearningUnits {
		if unit.IsPrimary {
			primaryCount++
		}
	}
	if primaryCount == 0 || primaryCount > 2 {
		t.Fatalf("expected one or two support primary units, got %#v", videos[0].LearningUnits)
	}
}

func TestDefaultVideoEvidenceAggregatorPreservesAllVideoLaneSources(t *testing.T) {
	aggregator := recommendationaggregator.NewDefaultVideoEvidenceAggregator()

	videos, err := aggregator.Aggregate(recommendationContext(), []model.ResolvedEvidenceWindow{
		resolvedWindow("video-5", 101, string(policy.BucketHardReview), string(policy.LaneExactCore), 0.70, 1000, 1450, 120_000, 0.34),
		resolvedWindow("video-5", 101, string(policy.BucketHardReview), string(policy.LaneBundle), 0.95, 2000, 2450, 120_000, 0.44),
	}, recommendationDemand())
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}

	if len(videos) != 1 {
		t.Fatalf("expected one video, got %#v", videos)
	}
	if !containsString(videos[0].LaneSources, string(policy.LaneExactCore)) || !containsString(videos[0].LaneSources, string(policy.LaneBundle)) {
		t.Fatalf("expected lane sources to include exact_core and bundle, got %#v", videos[0].LaneSources)
	}
}

func recommendationContext() model.RecommendationContext {
	return model.RecommendationContext{
		Request: model.RecommendationRequest{
			UserID:               "user-1",
			TargetVideoCount:     4,
			PreferredDurationSec: [2]int{45, 180},
		},
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func recommendationDemand() model.DemandBundle {
	return model.DemandBundle{
		HardReview: []model.DemandUnit{
			{UnitID: 101, Bucket: string(policy.BucketHardReview), Weight: 1.0},
		},
		NewNow: []model.DemandUnit{
			{UnitID: 201, Bucket: string(policy.BucketNewNow), Weight: 0.9},
			{UnitID: 202, Bucket: string(policy.BucketNewNow), Weight: 0.4},
		},
		SoftReview: []model.DemandUnit{
			{UnitID: 301, Bucket: string(policy.BucketSoftReview), Weight: 0.6},
			{UnitID: 302, Bucket: string(policy.BucketSoftReview), Weight: 0.55},
		},
		NearFuture: []model.DemandUnit{
			{UnitID: 401, Bucket: string(policy.BucketNearFuture), Weight: 0.5},
			{UnitID: 402, Bucket: string(policy.BucketNearFuture), Weight: 0.45},
		},
		TargetVideoCount: 4,
	}
}

func resolvedWindow(videoID string, unitID int64, bucket string, lane string, candidateScore float64, startMs int32, endMs int32, durationMs int32, coverageRatio float64) model.ResolvedEvidenceWindow {
	return model.ResolvedEvidenceWindow{
		Candidate: model.VideoUnitCandidate{
			VideoID:         videoID,
			CoarseUnitID:    unitID,
			Bucket:          bucket,
			Lane:            lane,
			UnitWeight:      candidateScore,
			CandidateScore:  candidateScore,
			CoverageRatio:   coverageRatio,
			DurationMs:      durationMs,
			MentionCount:    3,
			SentenceCount:   2,
			MappedSpanRatio: 0.7,
		},
		BestEvidenceRef:       &model.EvidenceRef{SentenceIndex: 1, SpanIndex: 1},
		BestEvidenceStartMs:   &startMs,
		BestEvidenceEndMs:     &endMs,
		WindowSentenceIndexes: []int32{1, 2},
		WindowStartMs:         &startMs,
		WindowEndMs:           &endMs,
		ResolvedSentences: []model.TranscriptSentence{
			{VideoID: videoID, SentenceIndex: 1, StartMs: startMs, EndMs: endMs},
		},
	}
}
