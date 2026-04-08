// 文件作用：
//   - 定义 QuotaAllocator，负责把 backlog 和请求上限映射成 review/new quota
//   - 当前实现承载 MVP 的配额分段规则和 backlog protection 开关
//
// 输入/输出：
//   - 输入：reviewBacklog、requestedLimit、RecommendationDefaults
//   - 输出：QuotaAllocation，包含 review/new 配额和 backlogProtection 标志
//
// 谁调用它：
//   - application/usecase/generate_recommendations.go
//   - unit test 会逐区间验证配额结果
//
// 它调用谁/传给谁：
//   - 调用本文件内的 ceilFraction 和 minInt
//   - 输出结果传给 RecommendationAssembler
package service

import (
	"math"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
)

type QuotaAllocation struct {
	ReviewQuota       int
	NewQuota          int
	BacklogProtection bool
}

type QuotaAllocator interface {
	Allocate(reviewBacklog, requestedLimit int, defaults model.RecommendationDefaults) QuotaAllocation
}

type quotaAllocator struct{}

func NewQuotaAllocator() QuotaAllocator {
	return quotaAllocator{}
}

func (quotaAllocator) Allocate(reviewBacklog, requestedLimit int, defaults model.RecommendationDefaults) QuotaAllocation {
	if requestedLimit < 0 {
		requestedLimit = 0
	}
	if reviewBacklog < 0 {
		reviewBacklog = 0
	}

	switch {
	case reviewBacklog == 0:
		return QuotaAllocation{
			ReviewQuota: 0,
			NewQuota:    minInt(requestedLimit, defaults.DailyNewUnitQuota),
		}
	case reviewBacklog > defaults.DailyReviewHardLimit:
		return QuotaAllocation{
			ReviewQuota:       minInt(requestedLimit, defaults.DailyReviewHardLimit),
			NewQuota:          0,
			BacklogProtection: true,
		}
	case reviewBacklog > defaults.DailyReviewSoftLimit:
		return QuotaAllocation{
			ReviewQuota: requestedLimit,
			NewQuota:    0,
		}
	case reviewBacklog >= 1 && reviewBacklog <= 5:
		reviewQuota := ceilFraction(requestedLimit, 0.5)
		return QuotaAllocation{
			ReviewQuota: reviewQuota,
			NewQuota:    minInt(requestedLimit-reviewQuota, defaults.DailyNewUnitQuota),
		}
	case reviewBacklog >= 6 && reviewBacklog <= 20:
		reviewQuota := ceilFraction(requestedLimit, 0.7)
		return QuotaAllocation{
			ReviewQuota: reviewQuota,
			NewQuota:    minInt(requestedLimit-reviewQuota, defaults.DailyNewUnitQuota),
		}
	case reviewBacklog >= 21 && reviewBacklog <= defaults.DailyReviewSoftLimit:
		reviewQuota := ceilFraction(requestedLimit, 0.85)
		return QuotaAllocation{
			ReviewQuota: reviewQuota,
			NewQuota:    minInt(3, requestedLimit-reviewQuota),
		}
	default:
		return QuotaAllocation{
			ReviewQuota: requestedLimit,
			NewQuota:    0,
		}
	}
}

func ceilFraction(limit int, ratio float64) int {
	return int(math.Ceil(float64(limit) * ratio))
}

func minInt(left, right int) int {
	if left < right {
		return left
	}

	return right
}
