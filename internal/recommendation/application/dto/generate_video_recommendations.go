package dto

type GenerateVideoRecommendationsRequest struct {
	UserID               string
	TargetVideoCount     int
	PreferredDurationSec [2]int
	SessionHint          string
	RequestContext       []byte
}

type RecommendationVideo struct {
	VideoID                   string
	Rank                      int
	Score                     float64
	ReasonCodes               []string
	CoveredUnits              []int64
	CoveredHardReviewUnits    []int64
	CoveredNewNowUnits        []int64
	CoveredSoftReviewUnits    []int64
	CoveredNearFutureUnits    []int64
	BestEvidenceSentenceIndex *int32
	BestEvidenceSpanIndex     *int32
	BestEvidenceStartMs       *int32
	BestEvidenceEndMs         *int32
	Explanation               string
}

type GenerateVideoRecommendationsResponse struct {
	RunID        string
	SelectorMode string
	Underfilled  bool
	Videos       []RecommendationVideo
}
