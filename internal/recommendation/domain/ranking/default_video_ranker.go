package ranking

import (
	"math"
	"sort"
	"time"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
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
		candidate.FreshnessScore = freshnessScore(videoServingByID[candidate.VideoID], videoUserByID[candidate.VideoID], candidate.DurationMs, recommendationContext.Now)
		candidate.RecentServedPenalty = recentServedPenalty(videoServingByID[candidate.VideoID], recommendationContext.Now)
		candidate.RecentWatchedPenalty = recentWatchedPenalty(videoUserByID[candidate.VideoID], candidate.DurationMs, recommendationContext.Now)
		candidate.OverloadPenalty = overloadPenalty(*candidate, recommendationContext.PreferredDurationSec)

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
		if rolePriority(ranked[i].DominantRole) != rolePriority(ranked[j].DominantRole) {
			return rolePriority(ranked[i].DominantRole) < rolePriority(ranked[j].DominantRole)
		}
		return ranked[i].VideoID < ranked[j].VideoID
	})

	_ = demand
	return ranked, nil
}

func freshnessScore(serving model.UserVideoServingState, watched model.VideoUserState, durationMs int32, now time.Time) float64 {
	freshness := 1.0
	if penalty := recentServedPenalty(serving, now); penalty > 0 {
		freshness -= penalty * 0.50
	}
	if penalty := recentWatchedPenalty(watched, durationMs, now); penalty > 0 {
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

func recentWatchedPenalty(state model.VideoUserState, durationMs int32, now time.Time) float64 {
	recency := 0.0
	if state.LastWatchedAt != nil {
		recency = recencyPenalty(now.Sub(*state.LastWatchedAt), 96*time.Hour)
	}
	countFactor := math.Min(float64(state.WatchCount)/5.0, 1.0)
	completionFactor := math.Min(float64(state.CompletedCount)/3.0, 1.0)
	return round4(math.Min(1.0, recency*0.45+countFactor*0.25+completionFactor*0.10+watchedRatio(state.MaxPositionMs, durationMs)*0.20))
}

func watchedRatio(maxPositionMs int32, durationMs int32) float64 {
	if maxPositionMs <= 0 || durationMs <= 0 {
		return 0
	}
	ratio := float64(maxPositionMs) / float64(durationMs)
	return math.Min(1.0, math.Max(0.0, ratio))
}

func overloadPenalty(candidate model.VideoCandidate, preferredDurationSec [2]int) float64 {
	coveredUnits := len(model.LearningUnitIDs(candidate.LearningUnits))
	penalty := 0.0
	if coveredUnits > 3 {
		penalty += math.Min(0.5, float64(coveredUnits-3)*0.25)
	}

	if evidence := model.PrimaryLearningUnitEvidence(candidate.LearningUnits); evidence != nil && evidence.StartMs != nil && evidence.EndMs != nil {
		if durationMs := *evidence.EndMs - *evidence.StartMs; durationMs > 0 {
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

func rolePriority(role model.LearningRole) int {
	switch role {
	case model.LearningRoleHardReview:
		return 0
	case model.LearningRoleNewNow:
		return 1
	case model.LearningRoleSoftReview:
		return 2
	default:
		return 3
	}
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}
