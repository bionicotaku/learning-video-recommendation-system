package dto

import "time"

type ListVideoFavoritesRequest struct {
	UserID string
	Limit  int
	Cursor string
}

type ListVideoHistoryRequest struct {
	UserID string
	Limit  int
	Cursor string
}

type ListVideoFavoritesResponse struct {
	Items []VideoFavoriteItem `json:"items"`
	Page  VideoLibraryPage    `json:"page"`
}

type ListVideoHistoryResponse struct {
	Items []VideoHistoryItem `json:"items"`
	Page  VideoLibraryPage   `json:"page"`
}

type VideoFavoriteItem struct {
	VideoID         string    `json:"video_id"`
	Title           string    `json:"title"`
	CoverImageURL   *string   `json:"cover_image_url"`
	DurationSeconds int       `json:"duration_seconds"`
	ViewCount       int64     `json:"view_count"`
	FavoritedAt     time.Time `json:"favorited_at"`
}

type VideoHistoryItem struct {
	VideoID         string    `json:"video_id"`
	Title           string    `json:"title"`
	CoverImageURL   *string   `json:"cover_image_url"`
	DurationSeconds int       `json:"duration_seconds"`
	ViewCount       int64     `json:"view_count"`
	LastPositionMS  int32     `json:"last_position_ms"`
	LastWatchedAt   time.Time `json:"last_watched_at"`
}

type VideoLibraryPage struct {
	Limit      int     `json:"limit"`
	HasMore    bool    `json:"has_more"`
	NextCursor *string `json:"next_cursor"`
}
