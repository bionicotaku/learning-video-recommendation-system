// 文件作用：
//   - 定义 PriorityZeroExtractor，从 scored review 中提取需要最高优先级出队的项目
//   - 当前规则聚焦于 learning 状态的 due review 和近期失败内容
//
// 输入/输出：
//   - 输入：ScoredReviewCandidate 列表
//   - 输出：按 priority-zero 规则筛选并排序后的 review 子集
//
// 谁调用它：
//   - application/usecase/generate_recommendations.go
//   - unit test 会直接验证筛选和排序行为
//
// 它调用谁/传给谁：
//   - 调用本文件内的 priorityZeroWeight 和 uniqueStrings
//   - 输出结果传给 RecommendationAssembler
package service

import (
	"sort"

	appquery "learning-video-recommendation-system/internal/recommendation/scheduler/application/query"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"
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
