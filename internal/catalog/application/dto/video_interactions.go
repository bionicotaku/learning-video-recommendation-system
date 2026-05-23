package dto

import "time"

type SetVideoLikeRequest struct {
	UserID     string
	VideoID    string
	Enabled    bool
	OccurredAt time.Time
}

type VideoLikeResponse struct {
	VideoID   string `json:"video_id"`
	HasLiked  bool   `json:"has_liked"`
	LikeCount int64  `json:"like_count"`
}

type SetVideoFavoriteRequest struct {
	UserID     string
	VideoID    string
	Enabled    bool
	OccurredAt time.Time
}

type VideoFavoriteResponse struct {
	VideoID       string `json:"video_id"`
	HasFavorited  bool   `json:"has_favorited"`
	FavoriteCount int64  `json:"favorite_count"`
}
