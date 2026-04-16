package selector

import (
	"math"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/domain/policy"
)

type DefaultVideoSelector struct{}

var _ VideoSelector = (*DefaultVideoSelector)(nil)

func NewDefaultVideoSelector() *DefaultVideoSelector {
	return &DefaultVideoSelector{}
}

func (s *DefaultVideoSelector) Select(recommendationContext model.RecommendationContext, ranked []model.VideoCandidate, demand model.DemandBundle) ([]model.VideoCandidate, error) {
	mode := selectorMode(demand)
	targetCount := recommendationContext.Request.TargetVideoCount
	if targetCount <= 0 {
		targetCount = demand.TargetVideoCount
	}
	if targetCount <= 0 {
		targetCount = 8
	}

	if mode == string(policy.SelectorModeExtremeSparse) {
		if len(ranked) < targetCount {
			return append([]model.VideoCandidate(nil), ranked...), nil
		}
		return append([]model.VideoCandidate(nil), ranked[:targetCount]...), nil
	}

	selected := make([]model.VideoCandidate, 0, minInt(targetCount, len(ranked)))
	selectedIDs := make(map[string]struct{}, len(ranked))
	dominantUnitCount := make(map[int64]int)
	coreCovered := make(map[int64]struct{})
	softCovered := make(map[int64]struct{})

	coreDominantMin := demand.MixQuota.CoreDominantMin
	if coreDominantMin == 0 {
		if mode == string(policy.SelectorModeLowSupply) {
			coreDominantMin = int(math.Ceil(float64(targetCount) * 0.375))
		} else {
			coreDominantMin = int(math.Ceil(float64(targetCount) * 0.50))
		}
	}

	for _, candidate := range ranked {
		if len(selected) >= targetCount || countCoreDominant(selected) >= coreDominantMin {
			break
		}
		if !isCoreDominant(candidate) {
			continue
		}
		if !canSelect(candidate, selected, demand, mode, dominantUnitCount) {
			continue
		}
		selected = append(selected, candidate)
		selectedIDs[candidate.VideoID] = struct{}{}
		incrementDominantUnitCount(candidate, dominantUnitCount)
		markCovered(candidate.CoveredHardReviewUnits, coreCovered)
		markCovered(candidate.CoveredNewNowUnits, coreCovered)
		markCovered(candidate.CoveredSoftReviewUnits, softCovered)
	}

	for len(selected) < targetCount {
		bestIndex := -1
		bestMarginalScore := -1.0
		for index, candidate := range ranked {
			if _, exists := selectedIDs[candidate.VideoID]; exists {
				continue
			}
			if !canSelect(candidate, selected, demand, mode, dominantUnitCount) {
				continue
			}

			score := marginalScore(candidate, coreCovered, softCovered, dominantUnitCount)
			if score > bestMarginalScore {
				bestMarginalScore = score
				bestIndex = index
			}
		}
		if bestIndex == -1 {
			break
		}

		candidate := ranked[bestIndex]
		selected = append(selected, candidate)
		selectedIDs[candidate.VideoID] = struct{}{}
		incrementDominantUnitCount(candidate, dominantUnitCount)
		markCovered(candidate.CoveredHardReviewUnits, coreCovered)
		markCovered(candidate.CoveredNewNowUnits, coreCovered)
		markCovered(candidate.CoveredSoftReviewUnits, softCovered)
	}

	return selected, nil
}

func selectorMode(demand model.DemandBundle) string {
	if demand.Flags.ExtremeSparse {
		return string(policy.SelectorModeExtremeSparse)
	}
	if demand.Flags.HardReviewLowSupply {
		return string(policy.SelectorModeLowSupply)
	}
	return string(policy.SelectorModeNormal)
}

func canSelect(candidate model.VideoCandidate, selected []model.VideoCandidate, demand model.DemandBundle, mode string, dominantUnitCount map[int64]int) bool {
	if candidate.DominantUnitID != nil && dominantUnitCount[*candidate.DominantUnitID] >= maxInt(1, demand.MixQuota.SameUnitMax) {
		return false
	}

	if isFallback(candidate) && countFallback(selected) >= maxInt(1, demand.MixQuota.FallbackMax) {
		return false
	}

	switch mode {
	case string(policy.SelectorModeNormal):
		if candidate.DominantBucket == string(policy.BucketNearFuture) && countFutureDominant(selected) >= demand.MixQuota.FutureDominantMax {
			return false
		}
	case string(policy.SelectorModeLowSupply):
		if isFutureLike(candidate) && countFutureLike(selected) >= demand.MixQuota.FutureLikeMax {
			return false
		}
	}

	return true
}

func marginalScore(candidate model.VideoCandidate, coreCovered map[int64]struct{}, softCovered map[int64]struct{}, dominantUnitCount map[int64]int) float64 {
	uncoveredCoreGain := countUncovered(candidate.CoveredHardReviewUnits, coreCovered) + countUncovered(candidate.CoveredNewNowUnits, coreCovered)
	uncoveredSoftGain := countUncovered(candidate.CoveredSoftReviewUnits, softCovered)
	redundancyPenalty := 0.0
	if uncoveredCoreGain == 0 && uncoveredSoftGain == 0 {
		redundancyPenalty = 0.08
	}

	repeatPenalty := 0.0
	if candidate.DominantUnitID != nil {
		repeatPenalty = float64(dominantUnitCount[*candidate.DominantUnitID]) * 0.04
	}

	return candidate.BaseScore + float64(uncoveredCoreGain)*0.10 + float64(uncoveredSoftGain)*0.04 - redundancyPenalty - repeatPenalty
}

func isCoreDominant(candidate model.VideoCandidate) bool {
	return candidate.DominantBucket == string(policy.BucketHardReview) || candidate.DominantBucket == string(policy.BucketNewNow)
}

func isFutureLike(candidate model.VideoCandidate) bool {
	return candidate.DominantBucket == string(policy.BucketSoftReview) || candidate.DominantBucket == string(policy.BucketNearFuture)
}

func isFallback(candidate model.VideoCandidate) bool {
	return len(candidate.LaneSources) == 1 && candidate.LaneSources[0] == string(policy.LaneQualityFallback)
}

func countCoreDominant(selected []model.VideoCandidate) int {
	count := 0
	for _, candidate := range selected {
		if isCoreDominant(candidate) {
			count++
		}
	}
	return count
}

func countFutureDominant(selected []model.VideoCandidate) int {
	count := 0
	for _, candidate := range selected {
		if candidate.DominantBucket == string(policy.BucketNearFuture) {
			count++
		}
	}
	return count
}

func countFutureLike(selected []model.VideoCandidate) int {
	count := 0
	for _, candidate := range selected {
		if isFutureLike(candidate) {
			count++
		}
	}
	return count
}

func countFallback(selected []model.VideoCandidate) int {
	count := 0
	for _, candidate := range selected {
		if isFallback(candidate) {
			count++
		}
	}
	return count
}

func incrementDominantUnitCount(candidate model.VideoCandidate, dominantUnitCount map[int64]int) {
	if candidate.DominantUnitID == nil {
		return
	}
	dominantUnitCount[*candidate.DominantUnitID]++
}

func markCovered(unitIDs []int64, covered map[int64]struct{}) {
	for _, unitID := range unitIDs {
		covered[unitID] = struct{}{}
	}
}

func countUncovered(unitIDs []int64, covered map[int64]struct{}) int {
	count := 0
	for _, unitID := range unitIDs {
		if _, ok := covered[unitID]; !ok {
			count++
		}
	}
	return count
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
