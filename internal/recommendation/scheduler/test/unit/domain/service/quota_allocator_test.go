package service_test

import (
	"testing"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	servicepkg "learning-video-recommendation-system/internal/recommendation/scheduler/domain/service"
)

func TestBacklogCalculator(t *testing.T) {
	calculator := servicepkg.NewBacklogCalculator()

	if got := calculator.Compute(12); got != 12 {
		t.Fatalf("Compute(12) = %d, want 12", got)
	}
	if got := calculator.Compute(-1); got != 0 {
		t.Fatalf("Compute(-1) = %d, want 0", got)
	}
}

func TestQuotaAllocatorRanges(t *testing.T) {
	allocator := servicepkg.NewQuotaAllocator()
	defaults := model.RecommendationDefaults{
		DailyNewUnitQuota:    8,
		DailyReviewSoftLimit: 30,
		DailyReviewHardLimit: 60,
	}

	tests := []struct {
		name            string
		reviewBacklog   int
		requestedLimit  int
		wantReviewQuota int
		wantNewQuota    int
		wantProtection  bool
	}{
		{name: "no backlog", reviewBacklog: 0, requestedLimit: 20, wantReviewQuota: 0, wantNewQuota: 8, wantProtection: false},
		{name: "small backlog lower bound", reviewBacklog: 1, requestedLimit: 20, wantReviewQuota: 10, wantNewQuota: 8, wantProtection: false},
		{name: "small backlog upper bound", reviewBacklog: 5, requestedLimit: 20, wantReviewQuota: 10, wantNewQuota: 8, wantProtection: false},
		{name: "medium backlog lower bound", reviewBacklog: 6, requestedLimit: 20, wantReviewQuota: 14, wantNewQuota: 6, wantProtection: false},
		{name: "medium backlog upper bound", reviewBacklog: 20, requestedLimit: 20, wantReviewQuota: 14, wantNewQuota: 6, wantProtection: false},
		{name: "soft backlog lower bound", reviewBacklog: 21, requestedLimit: 20, wantReviewQuota: 17, wantNewQuota: 3, wantProtection: false},
		{name: "soft backlog upper bound", reviewBacklog: 30, requestedLimit: 20, wantReviewQuota: 17, wantNewQuota: 3, wantProtection: false},
		{name: "over soft limit", reviewBacklog: 31, requestedLimit: 20, wantReviewQuota: 20, wantNewQuota: 0, wantProtection: false},
		{name: "over hard limit", reviewBacklog: 61, requestedLimit: 80, wantReviewQuota: 60, wantNewQuota: 0, wantProtection: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := allocator.Allocate(tt.reviewBacklog, tt.requestedLimit, defaults)
			if got.ReviewQuota != tt.wantReviewQuota {
				t.Fatalf("ReviewQuota = %d, want %d", got.ReviewQuota, tt.wantReviewQuota)
			}
			if got.NewQuota != tt.wantNewQuota {
				t.Fatalf("NewQuota = %d, want %d", got.NewQuota, tt.wantNewQuota)
			}
			if got.BacklogProtection != tt.wantProtection {
				t.Fatalf("BacklogProtection = %v, want %v", got.BacklogProtection, tt.wantProtection)
			}
		})
	}
}
