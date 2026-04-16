package dto

type GenerateVideoRecommendationsRequest struct {
	UserID               string
	TargetVideoCount     int
	PreferredDurationSec [2]int
	SessionHint          string
	RequestContext       []byte
}

type BestEvidence struct {
	SentenceIndex *int32
	SpanIndex     *int32
	StartMs       *int32
	EndMs         *int32
}

type RecommendationVideo struct {
	VideoID                string
	Rank                   int
	Score                  float64
	ReasonCodes            []string
	CoveredUnits           []int64
	CoveredHardReviewUnits []int64
	CoveredNewNowUnits     []int64
	CoveredSoftReviewUnits []int64
	CoveredNearFutureUnits []int64
	BestEvidence           *BestEvidence
	Explanation            string
}

type GenerateVideoRecommendationsResponse struct {
	RunID        string
	SelectorMode string
	Underfilled  bool
	Videos       []RecommendationVideo
}
