package dto

type GetVideoDetailRequest struct {
	UserID  string
	VideoID string
}

type VideoDetailResponse struct {
	VideoID         string               `json:"video_id"`
	Title           string               `json:"title"`
	Description     string               `json:"description"`
	VideoURL        string               `json:"video_url"`
	CoverImageURL   *string              `json:"cover_image_url"`
	TranscriptURL   *string              `json:"transcript_url"`
	DurationSeconds int                  `json:"duration_seconds"`
	ViewCount       int64                `json:"view_count"`
	LikeCount       int64                `json:"like_count"`
	FavoriteCount   int64                `json:"favorite_count"`
	UserState       VideoDetailUserState `json:"user_state"`
}

type VideoDetailUserState struct {
	HasLiked     bool `json:"has_liked"`
	HasFavorited bool `json:"has_favorited"`
}
