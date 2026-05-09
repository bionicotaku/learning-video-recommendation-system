package model

type FinalRecommendationItem struct {
	VideoID       string
	Rank          int
	Score         float64
	ReasonCodes   []string
	LearningUnits []ExpectedLearningUnit
	Explanation   string
}
