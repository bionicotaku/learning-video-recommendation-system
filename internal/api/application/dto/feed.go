package dto

type GetFeedRequest struct {
	UserID           string
	TargetVideoCount int
	ClientContext    []byte
}

type FeedResponse struct {
	RecommendationRunID string     `json:"recommendation_run_id"`
	Items               []FeedItem `json:"items"`
}

type FeedItem struct {
	VideoID         string             `json:"video_id"`
	Title           string             `json:"title"`
	Description     string             `json:"description"`
	VideoURL        string             `json:"video_url"`
	CoverImageURL   *string            `json:"cover_image_url"`
	TranscriptURL   *string            `json:"transcript_url"`
	DurationSeconds int                `json:"duration_seconds"`
	ViewCount       int64              `json:"view_count"`
	LikeCount       int64              `json:"like_count"`
	FavoriteCount   int64              `json:"favorite_count"`
	HasLiked        bool               `json:"has_liked"`
	HasFavorited    bool               `json:"has_favorited"`
	LearningUnits   []FeedLearningUnit `json:"learning_units"`
}

type FeedLearningUnit struct {
	CoarseUnitID          int64  `json:"coarse_unit_id"`
	Text                  string `json:"text"`
	Role                  string `json:"role"`
	IsPrimary             bool   `json:"is_primary"`
	EvidenceSentenceIndex int32  `json:"evidence_sentence_index"`
	EvidenceSpanIndex     int32  `json:"evidence_span_index"`
	EvidenceStartMS       int32  `json:"evidence_start_ms"`
	EvidenceEndMS         int32  `json:"evidence_end_ms"`
}
