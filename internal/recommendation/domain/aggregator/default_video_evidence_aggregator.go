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
		for _, window := range videoWindows {
			unitWindows[window.Candidate.CoarseUnitID] = append(unitWindows[window.Candidate.CoarseUnitID], window)
		}

		laneSources := make(map[string]struct{})
		coveredHard := make(map[int64]struct{})
		coveredNew := make(map[int64]struct{})
		coveredSoft := make(map[int64]struct{})
		coveredFuture := make(map[int64]struct{})

		bestVideoWindow := model.ResolvedEvidenceWindow{}
		bestVideoScore := -1.0
		var dominantUnitID *int64
		dominantBucket := ""

		totalCoverageStrength := 0.0
		totalEducationalFit := 0.0
		totalFutureValue := 0.0
		unitCount := 0

		for unitID, candidates := range unitWindows {
			sort.SliceStable(candidates, func(i, j int) bool {
				if evidenceStrength(candidates[i], recommendationContext.Request.PreferredDurationSec) != evidenceStrength(candidates[j], recommendationContext.Request.PreferredDurationSec) {
					return evidenceStrength(candidates[i], recommendationContext.Request.PreferredDurationSec) > evidenceStrength(candidates[j], recommendationContext.Request.PreferredDurationSec)
				}
				if candidates[i].BestEvidenceStartMs != nil && candidates[j].BestEvidenceStartMs != nil && *candidates[i].BestEvidenceStartMs != *candidates[j].BestEvidenceStartMs {
					return *candidates[i].BestEvidenceStartMs < *candidates[j].BestEvidenceStartMs
				}
				return candidates[i].Candidate.CoarseUnitID < candidates[j].Candidate.CoarseUnitID
			})

			best := candidates[0]
			unitStrength := evidenceStrength(best, recommendationContext.Request.PreferredDurationSec)
			if len(candidates) > 1 {
				unitStrength += evidenceStrength(candidates[1], recommendationContext.Request.PreferredDurationSec) * 0.15
			}
			unitStrength = math.Min(1, unitStrength)

			totalCoverageStrength += unitStrength
			totalEducationalFit += educationalFit(best, recommendationContext.Request.PreferredDurationSec)
			totalFutureValue += futureValue(best.Candidate.Bucket)
			unitCount++

			switch best.Candidate.Bucket {
			case string(policy.BucketHardReview):
				coveredHard[unitID] = struct{}{}
			case string(policy.BucketNewNow):
				coveredNew[unitID] = struct{}{}
			case string(policy.BucketSoftReview):
				coveredSoft[unitID] = struct{}{}
			case string(policy.BucketNearFuture):
				coveredFuture[unitID] = struct{}{}
			}

			laneSources[best.Candidate.Lane] = struct{}{}
			if pickAsBestVideoWindow(best, dominantBucket, bestVideoScore, unitStrength, bestVideoWindow, recommendationContext.Request.PreferredDurationSec) {
				bestVideoWindow = best
				bestVideoScore = unitStrength
				dominantBucket = best.Candidate.Bucket
				unitIDCopy := unitID
				dominantUnitID = &unitIDCopy
			}
		}

		distinctMatchedUnitCount := len(unitWindows)
		videos = append(videos, model.VideoCandidate{
			VideoID:                   videoID,
			LaneSources:               sortedKeys(laneSources),
			DominantBucket:            dominantBucket,
			DominantUnitID:            dominantUnitID,
			CoveredHardReviewUnits:    sortedInt64Keys(coveredHard),
			CoveredNewNowUnits:        sortedInt64Keys(coveredNew),
			CoveredSoftReviewUnits:    sortedInt64Keys(coveredSoft),
			CoveredNearFutureUnits:    sortedInt64Keys(coveredFuture),
			HardReviewCover:           coverageRatio(len(coveredHard), demandCounts[string(policy.BucketHardReview)]),
			NewNowCover:               coverageRatio(len(coveredNew), demandCounts[string(policy.BucketNewNow)]),
			SoftReviewCover:           coverageRatio(len(coveredSoft), demandCounts[string(policy.BucketSoftReview)]),
			NearFutureCover:           coverageRatio(len(coveredFuture), demandCounts[string(policy.BucketNearFuture)]),
			CoverageStrengthScore:     round4(totalCoverageStrength / float64(maxInt(1, unitCount))),
			BundleValueScore:          bundleValueScore(distinctMatchedUnitCount, len(coveredHard)+len(coveredNew) > 0, len(coveredSoft)+len(coveredFuture) > 0),
			EducationalFitScore:       round4(totalEducationalFit / float64(maxInt(1, unitCount))),
			FutureValueScore:          round4(totalFutureValue / float64(maxInt(1, unitCount))),
			BestEvidenceSentenceIndex: bestSentenceIndex(bestVideoWindow.BestEvidenceRef),
			BestEvidenceSpanIndex:     bestSpanIndex(bestVideoWindow.BestEvidenceRef),
			BestEvidenceStartMs:       bestVideoWindow.BestEvidenceStartMs,
			BestEvidenceEndMs:         bestVideoWindow.BestEvidenceEndMs,
		})
	}

	sort.SliceStable(videos, func(i, j int) bool {
		if bucketPriority(videos[i].DominantBucket) != bucketPriority(videos[j].DominantBucket) {
			return bucketPriority(videos[i].DominantBucket) < bucketPriority(videos[j].DominantBucket)
		}
		if videos[i].CoverageStrengthScore != videos[j].CoverageStrengthScore {
			return videos[i].CoverageStrengthScore > videos[j].CoverageStrengthScore
		}
		return videos[i].VideoID < videos[j].VideoID
	})

	return videos, nil
}

func pickAsBestVideoWindow(candidate model.ResolvedEvidenceWindow, currentBucket string, currentScore float64, candidateScore float64, current model.ResolvedEvidenceWindow, preferredDurationSec [2]int) bool {
	if currentScore < 0 {
		return true
	}
	if bucketPriority(candidate.Candidate.Bucket) != bucketPriority(currentBucket) {
		return bucketPriority(candidate.Candidate.Bucket) < bucketPriority(currentBucket)
	}
	if candidateScore != currentScore {
		return candidateScore > currentScore
	}
	return bestStart(candidate) < bestStart(current)
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
