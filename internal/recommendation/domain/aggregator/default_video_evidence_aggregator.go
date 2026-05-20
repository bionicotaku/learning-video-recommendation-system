package aggregator

import (
	"math"
	"sort"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/domain/policy"
)

type DefaultVideoEvidenceAggregator struct{}

var _ VideoEvidenceAggregator = (*DefaultVideoEvidenceAggregator)(nil)

func NewDefaultVideoEvidenceAggregator() *DefaultVideoEvidenceAggregator {
	return &DefaultVideoEvidenceAggregator{}
}

func (a *DefaultVideoEvidenceAggregator) Aggregate(recommendationContext model.RecommendationContext, windows []model.ResolvedEvidenceWindow, demand model.DemandBundle) ([]model.VideoCandidate, error) {
	demandCounts := bucketDemandCounts(demand)
	grouped := make(map[string][]model.ResolvedEvidenceWindow)
	for _, window := range windows {
		grouped[window.Candidate.VideoID] = append(grouped[window.Candidate.VideoID], window)
	}

	videos := make([]model.VideoCandidate, 0, len(grouped))
	for videoID, videoWindows := range grouped {
		unitWindows := make(map[int64][]model.ResolvedEvidenceWindow)
		laneSources := make(map[string]struct{})
		for _, window := range videoWindows {
			unitWindows[window.Candidate.CoarseUnitID] = append(unitWindows[window.Candidate.CoarseUnitID], window)
			laneSources[window.Candidate.Lane] = struct{}{}
		}

		bestVideoScore := -1.0
		dominantStartMs := int32(math.MaxInt32)
		var dominantUnitID *int64
		dominantRole := model.LearningRole("")

		learningUnitCandidates := make([]aggregatedLearningUnit, 0, len(unitWindows))
		preferredDurationSec := recommendationContext.PreferredDurationSec

		for unitID, candidates := range unitWindows {
			sort.SliceStable(candidates, func(i, j int) bool {
				if evidenceStrength(candidates[i], preferredDurationSec) != evidenceStrength(candidates[j], preferredDurationSec) {
					return evidenceStrength(candidates[i], preferredDurationSec) > evidenceStrength(candidates[j], preferredDurationSec)
				}
				if candidates[i].BestEvidenceStartMs != nil && candidates[j].BestEvidenceStartMs != nil && *candidates[i].BestEvidenceStartMs != *candidates[j].BestEvidenceStartMs {
					return *candidates[i].BestEvidenceStartMs < *candidates[j].BestEvidenceStartMs
				}
				return candidates[i].Candidate.CoarseUnitID < candidates[j].Candidate.CoarseUnitID
			})

			best := candidates[0]
			role := roleFromBucket(best.Candidate.Bucket)
			unitStrength := evidenceStrength(best, preferredDurationSec)
			if len(candidates) > 1 {
				unitStrength += evidenceStrength(candidates[1], preferredDurationSec) * 0.15
			}
			unitStrength = math.Min(1, unitStrength)

			if pickAsDominantLearningUnit(best, dominantRole, bestVideoScore, dominantStartMs, unitStrength, preferredDurationSec) {
				bestVideoScore = unitStrength
				dominantStartMs = bestStart(best)
				dominantRole = role
				unitIDCopy := unitID
				dominantUnitID = &unitIDCopy
			}
			learningUnitCandidates = append(learningUnitCandidates, aggregatedLearningUnit{
				unit: model.ExpectedLearningUnit{
					CoarseUnitID: unitID,
					Role:         role,
					Evidence:     evidenceFromWindow(best),
				},
				strength:       unitStrength,
				educationalFit: educationalFit(best, preferredDurationSec),
				futureValue:    futureValue(best.Candidate.Bucket),
				startMs:        bestStart(best),
			})
		}

		sort.SliceStable(learningUnitCandidates, func(i, j int) bool {
			if rolePriority(learningUnitCandidates[i].unit.Role) != rolePriority(learningUnitCandidates[j].unit.Role) {
				return rolePriority(learningUnitCandidates[i].unit.Role) < rolePriority(learningUnitCandidates[j].unit.Role)
			}
			if learningUnitCandidates[i].strength != learningUnitCandidates[j].strength {
				return learningUnitCandidates[i].strength > learningUnitCandidates[j].strength
			}
			if learningUnitCandidates[i].startMs != learningUnitCandidates[j].startMs {
				return learningUnitCandidates[i].startMs < learningUnitCandidates[j].startMs
			}
			return learningUnitCandidates[i].unit.CoarseUnitID < learningUnitCandidates[j].unit.CoarseUnitID
		})
		if len(learningUnitCandidates) > 8 {
			learningUnitCandidates = learningUnitCandidates[:8]
		}
		learningUnits := finalizePrimaryLearningUnits(learningUnitCandidates)
		totalCoverageStrength := 0.0
		totalEducationalFit := 0.0
		totalFutureValue := 0.0
		for _, candidate := range learningUnitCandidates {
			totalCoverageStrength += candidate.strength
			totalEducationalFit += candidate.educationalFit
			totalFutureValue += candidate.futureValue
		}
		unitCount := maxInt(1, len(learningUnits))
		hasCore := hasAnyRole(learningUnits, model.LearningRoleHardReview, model.LearningRoleNewNow)
		hasSupportingUnits := hasAnyRole(learningUnits, model.LearningRoleSoftReview, model.LearningRoleNearFuture)

		videos = append(videos, model.VideoCandidate{
			VideoID:               videoID,
			DurationMs:            videoWindows[0].Candidate.DurationMs,
			LaneSources:           sortedKeys(laneSources),
			DominantRole:          dominantRole,
			DominantUnitID:        dominantUnitID,
			LearningUnits:         learningUnits,
			HardReviewCover:       coverageRatio(model.CountLearningUnitsByRole(learningUnits, model.LearningRoleHardReview), demandCounts[string(policy.BucketHardReview)]),
			NewNowCover:           coverageRatio(model.CountLearningUnitsByRole(learningUnits, model.LearningRoleNewNow), demandCounts[string(policy.BucketNewNow)]),
			SoftReviewCover:       coverageRatio(model.CountLearningUnitsByRole(learningUnits, model.LearningRoleSoftReview), demandCounts[string(policy.BucketSoftReview)]),
			NearFutureCover:       coverageRatio(model.CountLearningUnitsByRole(learningUnits, model.LearningRoleNearFuture), demandCounts[string(policy.BucketNearFuture)]),
			CoverageStrengthScore: round4(totalCoverageStrength / float64(unitCount)),
			BundleValueScore:      bundleValueScore(len(learningUnits), hasCore, hasSupportingUnits),
			EducationalFitScore:   round4(totalEducationalFit / float64(unitCount)),
			FutureValueScore:      round4(totalFutureValue / float64(unitCount)),
		})
	}

	sort.SliceStable(videos, func(i, j int) bool {
		if rolePriority(videos[i].DominantRole) != rolePriority(videos[j].DominantRole) {
			return rolePriority(videos[i].DominantRole) < rolePriority(videos[j].DominantRole)
		}
		if videos[i].CoverageStrengthScore != videos[j].CoverageStrengthScore {
			return videos[i].CoverageStrengthScore > videos[j].CoverageStrengthScore
		}
		return videos[i].VideoID < videos[j].VideoID
	})

	return videos, nil
}

type aggregatedLearningUnit struct {
	unit           model.ExpectedLearningUnit
	strength       float64
	educationalFit float64
	futureValue    float64
	startMs        int32
}

func pickAsDominantLearningUnit(candidate model.ResolvedEvidenceWindow, currentRole model.LearningRole, currentScore float64, currentStartMs int32, candidateScore float64, preferredDurationSec [2]int) bool {
	if currentScore < 0 {
		return true
	}
	candidateRole := roleFromBucket(candidate.Candidate.Bucket)
	if rolePriority(candidateRole) != rolePriority(currentRole) {
		return rolePriority(candidateRole) < rolePriority(currentRole)
	}
	if candidateScore != currentScore {
		return candidateScore > currentScore
	}
	return bestStart(candidate) < currentStartMs
}

func evidenceStrength(window model.ResolvedEvidenceWindow, preferredDurationSec [2]int) float64 {
	return round4(
		window.Candidate.CandidateScore*0.55 +
			window.Candidate.CoverageRatio*0.20 +
			window.Candidate.MappedSpanRatio*0.15 +
			evidenceFocus(window)*0.10 +
			durationFit(window.Candidate.DurationMs, preferredDurationSec)*0.05,
	)
}

func evidenceFocus(window model.ResolvedEvidenceWindow) float64 {
	if window.WindowStartMs == nil || window.WindowEndMs == nil || window.Candidate.DurationMs <= 0 {
		return 0.5
	}
	windowDuration := *window.WindowEndMs - *window.WindowStartMs
	if windowDuration <= 0 {
		return 0.5
	}
	focus := float64(windowDuration) / float64(window.Candidate.DurationMs)
	return round4(math.Max(0.0, math.Min(1.0, 1.0-focus)))
}

func educationalFit(window model.ResolvedEvidenceWindow, preferredDurationSec [2]int) float64 {
	windowCompleteness := 0.5
	if len(window.WindowSentenceIndexes) > 0 && len(window.ResolvedSentences) > 0 {
		windowCompleteness = math.Min(1.0, float64(len(window.ResolvedSentences))/float64(len(window.WindowSentenceIndexes)))
	}
	return round4(durationFit(window.Candidate.DurationMs, preferredDurationSec)*0.55 + windowCompleteness*0.45)
}

func futureValue(bucket string) float64 {
	switch bucket {
	case string(policy.BucketNearFuture):
		return 1.0
	case string(policy.BucketSoftReview):
		return 0.6
	default:
		return 0.2
	}
}

func bundleValueScore(distinctMatchedUnitCount int, hasCore bool, hasSupportingUnits bool) float64 {
	score := 0.0
	if distinctMatchedUnitCount > 1 {
		score += math.Min(1.0, float64(distinctMatchedUnitCount-1)/3.0)
	}
	if hasCore {
		score += 0.20
	}
	if hasSupportingUnits {
		score += 0.10
	}
	return round4(math.Min(1.0, score))
}

func durationFit(durationMs int32, preferredDurationSec [2]int) float64 {
	minMs := preferredDurationSec[0] * 1000
	maxMs := preferredDurationSec[1] * 1000
	if int(durationMs) >= minMs && int(durationMs) <= maxMs {
		return 1.0
	}
	if int(durationMs) < minMs {
		delta := float64(minMs-int(durationMs)) / float64(minMs)
		return round4(math.Max(0.0, 1.0-delta))
	}
	delta := float64(int(durationMs)-maxMs) / float64(maxMs)
	return round4(math.Max(0.0, 1.0-delta))
}

func bucketDemandCounts(demand model.DemandBundle) map[string]int {
	return map[string]int{
		string(policy.BucketHardReview): len(demand.HardReview),
		string(policy.BucketNewNow):     len(demand.NewNow),
		string(policy.BucketSoftReview): len(demand.SoftReview),
		string(policy.BucketNearFuture): len(demand.NearFuture),
	}
}

func coverageRatio(covered int, total int) float64 {
	if total == 0 {
		return 0
	}
	return round4(float64(covered) / float64(total))
}

func sortedKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func sortedInt64Keys(values map[int64]struct{}) []int64 {
	result := make([]int64, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	return result
}

func evidenceFromWindow(window model.ResolvedEvidenceWindow) *model.LearningUnitEvidence {
	evidence := &model.LearningUnitEvidence{
		SentenceIndex: bestSentenceIndex(window.BestEvidenceRef),
		SpanIndex:     bestSpanIndex(window.BestEvidenceRef),
		StartMs:       window.BestEvidenceStartMs,
		EndMs:         window.BestEvidenceEndMs,
	}
	if evidence.SentenceIndex == nil && evidence.SpanIndex == nil && evidence.StartMs == nil && evidence.EndMs == nil {
		return nil
	}
	return evidence
}

func bestSentenceIndex(ref *model.EvidenceRef) *int32 {
	if ref == nil {
		return nil
	}
	index := ref.SentenceIndex
	return &index
}

func bestSpanIndex(ref *model.EvidenceRef) *int32 {
	if ref == nil {
		return nil
	}
	index := ref.SpanIndex
	return &index
}

func bestStart(window model.ResolvedEvidenceWindow) int32 {
	if window.BestEvidenceStartMs == nil {
		return math.MaxInt32
	}
	return *window.BestEvidenceStartMs
}

func roleFromBucket(bucket string) model.LearningRole {
	switch bucket {
	case string(policy.BucketHardReview):
		return model.LearningRoleHardReview
	case string(policy.BucketNewNow):
		return model.LearningRoleNewNow
	case string(policy.BucketSoftReview):
		return model.LearningRoleSoftReview
	default:
		return model.LearningRoleNearFuture
	}
}

func finalizePrimaryLearningUnits(candidates []aggregatedLearningUnit) []model.ExpectedLearningUnit {
	result := make([]model.ExpectedLearningUnit, 0, len(candidates))
	hasCore := false
	for _, candidate := range candidates {
		unit := candidate.unit
		if model.IsCoreLearningRole(unit.Role) {
			unit.IsPrimary = true
			hasCore = true
		}
		result = append(result, unit)
	}
	if hasCore {
		return result
	}

	primaryCount := 0
	for index := range result {
		if primaryCount >= 2 {
			break
		}
		if model.IsFutureLikeLearningRole(result[index].Role) {
			result[index].IsPrimary = true
			primaryCount++
		}
	}
	return result
}

func hasAnyRole(units []model.ExpectedLearningUnit, roles ...model.LearningRole) bool {
	for _, role := range roles {
		if model.HasLearningRole(units, role) {
			return true
		}
	}
	return false
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
