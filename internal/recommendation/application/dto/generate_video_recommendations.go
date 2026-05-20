package dto

type GenerateVideoRecommendationsRequest struct {
	UserID           string `json:"user_id"`
	TargetVideoCount int    `json:"target_video_count"`
	RequestContext   []byte `json:"request_context"`
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

type RecommendationPlanItem struct {
	VideoID       string                 `json:"video_id"`
	DurationMs    int32                  `json:"duration_ms"`
	LearningUnits []ExpectedLearningUnit `json:"learning_units"`
}

type GenerateVideoRecommendationsResponse struct {
	RunID string                   `json:"run_id"`
	Items []RecommendationPlanItem `json:"items"`
}
