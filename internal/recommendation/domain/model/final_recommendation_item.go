package model

type FinalRecommendationItem struct {
	VideoID       string
	DurationMs    int32
	Score         float64
	ReasonCodes   []string
	LearningUnits []ExpectedLearningUnit
}
