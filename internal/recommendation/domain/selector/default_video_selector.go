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

	selected := make([]model.VideoCandidate, 0, minInt(targetCount, len(ranked)))
	selectedIDs := make(map[string]struct{}, len(ranked))
	primaryUnitCount := make(map[int64]int)
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
		if !canSelect(candidate, selected, demand, mode, primaryUnitCount) {
			continue
		}
		selected = append(selected, candidate)
		selectedIDs[candidate.VideoID] = struct{}{}
		markPrimaryUnitsSelected(candidate, primaryUnitCount)
		markCovered(model.LearningUnitIDsByRoles(candidate.LearningUnits, model.LearningRoleHardReview, model.LearningRoleNewNow), coreCovered)
		markCovered(model.LearningUnitIDsByRoles(candidate.LearningUnits, model.LearningRoleSoftReview), softCovered)
	}

	for len(selected) < targetCount {
		bestIndex := -1
		bestMarginalScore := -1.0
		for index, candidate := range ranked {
			if _, exists := selectedIDs[candidate.VideoID]; exists {
				continue
			}
			if !canSelect(candidate, selected, demand, mode, primaryUnitCount) {
				continue
			}

			score := marginalScore(candidate, coreCovered, softCovered, primaryUnitCount)
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
		markPrimaryUnitsSelected(candidate, primaryUnitCount)
		markCovered(model.LearningUnitIDsByRoles(candidate.LearningUnits, model.LearningRoleHardReview, model.LearningRoleNewNow), coreCovered)
		markCovered(model.LearningUnitIDsByRoles(candidate.LearningUnits, model.LearningRoleSoftReview), softCovered)
	}

	return selected, nil
}

func selectorMode(demand model.DemandBundle) string {
	if demand.Flags.HardReviewLowSupply {
		return string(policy.SelectorModeLowSupply)
	}
	return string(policy.SelectorModeNormal)
}

func canSelect(candidate model.VideoCandidate, selected []model.VideoCandidate, demand model.DemandBundle, mode string, primaryUnitCount map[int64]int) bool {
	sameUnitMax := maxInt(1, demand.MixQuota.SameUnitMax)
	for _, unitID := range model.PrimaryLearningUnitIDs(candidate.LearningUnits) {
		if primaryUnitCount[unitID] >= sameUnitMax {
			return false
		}
	}

	if isFallback(candidate) && countFallback(selected) >= maxInt(1, demand.MixQuota.FallbackMax) {
		return false
	}

	switch mode {
	case string(policy.SelectorModeNormal):
		if candidate.DominantRole == model.LearningRoleNearFuture && countFutureDominant(selected) >= demand.MixQuota.FutureDominantMax {
			return false
		}
	case string(policy.SelectorModeLowSupply):
		if isFutureLike(candidate) && countFutureLike(selected) >= demand.MixQuota.FutureLikeMax {
			return false
		}
	}

	return true
}

func marginalScore(candidate model.VideoCandidate, coreCovered map[int64]struct{}, softCovered map[int64]struct{}, primaryUnitCount map[int64]int) float64 {
	uncoveredCoreGain := countUncovered(model.LearningUnitIDsByRoles(candidate.LearningUnits, model.LearningRoleHardReview, model.LearningRoleNewNow), coreCovered)
	uncoveredSoftGain := countUncovered(model.LearningUnitIDsByRoles(candidate.LearningUnits, model.LearningRoleSoftReview), softCovered)
	redundancyPenalty := 0.0
	if uncoveredCoreGain == 0 && uncoveredSoftGain == 0 {
		redundancyPenalty = 0.08
	}

	repeatPenalty := 0.0
	for _, unitID := range model.PrimaryLearningUnitIDs(candidate.LearningUnits) {
		repeatPenalty += float64(primaryUnitCount[unitID]) * 0.04
	}

	return candidate.BaseScore + float64(uncoveredCoreGain)*0.10 + float64(uncoveredSoftGain)*0.04 - redundancyPenalty - repeatPenalty
}

func isCoreDominant(candidate model.VideoCandidate) bool {
	return model.IsCoreLearningRole(candidate.DominantRole)
}

func isFutureLike(candidate model.VideoCandidate) bool {
	return model.IsFutureLikeLearningRole(candidate.DominantRole)
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
		if candidate.DominantRole == model.LearningRoleNearFuture {
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

func markPrimaryUnitsSelected(candidate model.VideoCandidate, primaryUnitCount map[int64]int) {
	for _, unitID := range model.PrimaryLearningUnitIDs(candidate.LearningUnits) {
		primaryUnitCount[unitID]++
	}
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
