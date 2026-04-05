package enum

// RecommendType identifies whether a recommendation item is review or new.
type RecommendType string

const (
	RecommendTypeReview RecommendType = "review"
	RecommendTypeNew    RecommendType = "new"
)
