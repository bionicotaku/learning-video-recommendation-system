package service

import (
	"math"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
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
