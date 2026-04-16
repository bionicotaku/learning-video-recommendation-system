package ranking

import (
	"math"
	"sort"
	"time"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/domain/policy"
)

type DefaultVideoRanker struct{}

var _ VideoRanker = (*DefaultVideoRanker)(nil)

func NewDefaultVideoRanker() *DefaultVideoRanker {
	return &DefaultVideoRanker{}
}

func (r *DefaultVideoRanker) Rank(recommendationContext model.RecommendationContext, candidates []model.VideoCandidate, demand model.DemandBundle) ([]model.VideoCandidate, error) {
	videoServingByID := make(map[string]model.UserVideoServingState, len(recommendationContext.VideoServingStates))
	for _, state := range recommendationContext.VideoServingStates {
		videoServingByID[state.VideoID] = state
	}
	videoUserByID := make(map[string]model.VideoUserState, len(recommendationContext.VideoUserStates))
	for _, state := range recommendationContext.VideoUserStates {
		videoUserByID[state.VideoID] = state
	}

	ranked := append([]model.VideoCandidate(nil), candidates...)
	for index := range ranked {
		candidate := &ranked[index]
		candidate.FreshnessScore = freshnessScore(videoServingByID[candidate.VideoID], videoUserByID[candidate.VideoID], recommendationContext.Now)
		candidate.RecentServedPenalty = recentServedPenalty(videoServingByID[candidate.VideoID], recommendationContext.Now)
		candidate.RecentWatchedPenalty = recentWatchedPenalty(videoUserByID[candidate.VideoID], recommendationContext.Now)
		candidate.OverloadPenalty = overloadPenalty(*candidate, recommendationContext.Request.PreferredDurationSec)

		demandCoverage := 0.50*candidate.HardReviewCover +
			0.20*candidate.NewNowCover +
			0.20*candidate.SoftReviewCover +
			0.10*candidate.NearFutureCover

		candidate.BaseScore = round4(
			0.40*demandCoverage +
				0.18*candidate.CoverageStrengthScore +
				0.15*candidate.BundleValueScore +
				0.15*candidate.EducationalFitScore +
				0.05*candidate.FutureValueScore +
				0.05*candidate.FreshnessScore -
				0.03*candidate.RecentServedPenalty -
				0.02*candidate.OverloadPenalty,
		)
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].BaseScore != ranked[j].BaseScore {
			return ranked[i].BaseScore > ranked[j].BaseScore
		}
		if bucketPriority(ranked[i].DominantBucket) != bucketPriority(ranked[j].DominantBucket) {
			return bucketPriority(ranked[i].DominantBucket) < bucketPriority(ranked[j].DominantBucket)
		}
		return ranked[i].VideoID < ranked[j].VideoID
	})

	_ = demand
	return ranked, nil
}

func freshnessScore(serving model.UserVideoServingState, watched model.VideoUserState, now time.Time) float64 {
	freshness := 1.0
	if penalty := recentServedPenalty(serving, now); penalty > 0 {
		freshness -= penalty * 0.50
	}
	if penalty := recentWatchedPenalty(watched, now); penalty > 0 {
		freshness -= penalty * 0.35
	}
	return round4(math.Max(0, freshness))
}

func recentServedPenalty(state model.UserVideoServingState, now time.Time) float64 {
	if state.LastServedAt == nil {
		return 0
	}
	recency := recencyPenalty(now.Sub(*state.LastServedAt), 72*time.Hour)
	countFactor := math.Min(float64(state.ServedCount)/5.0, 1.0)
	return round4(math.Min(1.0, recency*0.70+countFactor*0.30))
}

func recentWatchedPenalty(state model.VideoUserState, now time.Time) float64 {
	recency := 0.0
	if state.LastWatchedAt != nil {
		recency = recencyPenalty(now.Sub(*state.LastWatchedAt), 96*time.Hour)
	}
	countFactor := math.Min(float64(state.WatchCount)/5.0, 1.0)
	completionFactor := math.Min(float64(state.CompletedCount)/3.0, 1.0)
	return round4(math.Min(1.0, recency*0.45+countFactor*0.25+completionFactor*0.10+state.MaxWatchRatio*0.20))
}

func overloadPenalty(candidate model.VideoCandidate, preferredDurationSec [2]int) float64 {
	coveredUnits := len(uniqueCoveredUnits(candidate))
	penalty := 0.0
	if coveredUnits > 3 {
		penalty += math.Min(0.5, float64(coveredUnits-3)*0.25)
	}

	if candidate.BestEvidenceStartMs != nil && candidate.BestEvidenceEndMs != nil {
		if durationMs := *candidate.BestEvidenceEndMs - *candidate.BestEvidenceStartMs; durationMs > 0 {
			windowRatio := float64(durationMs) / float64(preferredDurationSec[1]*1000)
			if windowRatio > 1 {
				penalty += math.Min(0.25, windowRatio-1)
			}
		}
	}

	return round4(math.Min(1.0, penalty))
}

func recencyPenalty(delta time.Duration, horizon time.Duration) float64 {
	if delta <= 0 {
		return 1.0
	}
	if delta >= horizon {
		return 0
	}
	return round4(1.0 - (float64(delta) / float64(horizon)))
}

func uniqueCoveredUnits(candidate model.VideoCandidate) []int64 {
	seen := make(map[int64]struct{})
	result := make([]int64, 0)
	for _, units := range [][]int64{
		candidate.CoveredHardReviewUnits,
		candidate.CoveredNewNowUnits,
		candidate.CoveredSoftReviewUnits,
		candidate.CoveredNearFutureUnits,
	} {
		for _, unitID := range units {
			if _, ok := seen[unitID]; ok {
				continue
			}
			seen[unitID] = struct{}{}
			result = append(result, unitID)
		}
	}
	return result
}

func bucketPriority(bucket string) int {
	switch bucket {
	case string(policy.BucketHardReview):
		return 0
	case string(policy.BucketNewNow):
		return 1
	case string(policy.BucketSoftReview):
		return 2
	default:
		return 3
	}
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}
