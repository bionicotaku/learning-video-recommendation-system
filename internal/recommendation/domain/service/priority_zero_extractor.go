package service

import (
	"sort"

	appquery "learning-video-recommendation-system/internal/recommendation/application/query"
	"learning-video-recommendation-system/internal/recommendation/domain/enum"
)

type PriorityZeroExtractor interface {
	Extract(scoredReviews []appquery.ScoredReviewCandidate) []appquery.ScoredReviewCandidate
}

type priorityZeroExtractor struct{}

func NewPriorityZeroExtractor() PriorityZeroExtractor {
	return priorityZeroExtractor{}
}

func (priorityZeroExtractor) Extract(scoredReviews []appquery.ScoredReviewCandidate) []appquery.ScoredReviewCandidate {
	items := make([]appquery.ScoredReviewCandidate, 0)
	for _, item := range scoredReviews {
		isLearningDue := item.Candidate.State.Status == enum.UnitStatusLearning
		isRecentFailure := item.Candidate.State.LastQuality != nil && *item.Candidate.State.LastQuality <= 2
		if !isLearningDue && !isRecentFailure {
			continue
		}

		reasonCodes := append([]string{}, item.ReasonCodes...)
		if isLearningDue {
			reasonCodes = append(reasonCodes, "priority_zero_learning_due")
		}
		if isRecentFailure {
			reasonCodes = append(reasonCodes, "priority_zero_recent_failure")
		}

		item.ReasonCodes = uniqueStrings(reasonCodes)
		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		leftPriority := priorityZeroWeight(items[i])
		rightPriority := priorityZeroWeight(items[j])
		if leftPriority == rightPriority {
			if items[i].Score == items[j].Score {
				return items[i].Candidate.State.CoarseUnitID < items[j].Candidate.State.CoarseUnitID
			}

			return items[i].Score > items[j].Score
		}

		return leftPriority > rightPriority
	})

	return items
}

func priorityZeroWeight(item appquery.ScoredReviewCandidate) int {
	weight := 0
	if item.Candidate.State.Status == enum.UnitStatusLearning {
		weight += 2
	}
	if item.Candidate.State.LastQuality != nil && *item.Candidate.State.LastQuality <= 2 {
		weight++
	}

	return weight
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}
