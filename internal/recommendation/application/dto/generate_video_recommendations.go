package dto

type GenerateVideoRecommendationsRequest struct {
	UserID               string `json:"user_id"`
	TargetVideoCount     int    `json:"target_video_count"`
	PreferredDurationSec [2]int `json:"preferred_duration_sec"`
	SessionHint          string `json:"session_hint"`
	RequestContext       []byte `json:"request_context"`
}

type LearningUnitEvidence struct {
	SentenceIndex *int32 `json:"sentence_index,omitempty"`
	SpanIndex     *int32 `json:"span_index,omitempty"`
	StartMs       *int32 `json:"start_ms,omitempty"`
	EndMs         *int32 `json:"end_ms,omitempty"`
}

type ExpectedLearningUnit struct {
	CoarseUnitID int64                 `json:"coarse_unit_id"`
	Role         string                `json:"role"`
	IsPrimary    bool                  `json:"is_primary"`
	Evidence     *LearningUnitEvidence `json:"evidence,omitempty"`
}

type RecommendationVideo struct {
	VideoID       string                 `json:"video_id"`
	Rank          int                    `json:"rank"`
	Score         float64                `json:"score"`
	ReasonCodes   []string               `json:"reason_codes"`
	LearningUnits []ExpectedLearningUnit `json:"learning_units"`
	Explanation   string                 `json:"explanation"`
}

type GenerateVideoRecommendationsResponse struct {
	RunID        string                `json:"run_id"`
	SelectorMode string                `json:"selector_mode"`
	Underfilled  bool                  `json:"underfilled"`
	Videos       []RecommendationVideo `json:"videos"`
}
