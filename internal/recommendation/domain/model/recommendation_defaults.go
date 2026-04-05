package model

// RecommendationDefaults contains the fixed MVP recommendation limits.
type RecommendationDefaults struct {
	SessionDefaultLimit  int
	DailyNewUnitQuota    int
	DailyReviewSoftLimit int
	DailyReviewHardLimit int
	Timezone             string
}

// DefaultRecommendationDefaults returns the fixed MVP recommendation defaults.
func DefaultRecommendationDefaults() RecommendationDefaults {
	return RecommendationDefaults{
		SessionDefaultLimit:  20,
		DailyNewUnitQuota:    8,
		DailyReviewSoftLimit: 30,
		DailyReviewHardLimit: 60,
	}
}
