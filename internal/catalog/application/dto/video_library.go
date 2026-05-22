package dto

import "time"

const (
	VideoLibraryCursorKindFavorites = "video_favorites"
	VideoLibraryCursorKindHistory   = "video_history"
)

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

type ListVideoFavoritesQuery struct {
	UserID       string
	LimitPlusOne int
	Cursor       *VideoLibraryCursor
}

type ListVideoHistoryQuery struct {
	UserID       string
	LimitPlusOne int
	Cursor       *VideoLibraryCursor
}

type ListVideoFavoritesResponse struct {
	Items []VideoFavoriteItem
	Page  VideoLibraryPage
}

type ListVideoHistoryResponse struct {
	Items []VideoHistoryItem
	Page  VideoLibraryPage
}

type VideoFavoriteItem struct {
	VideoID       string
	Title         string
	CoverImageURL *string
	DurationMS    int32
	ViewCount     int64
	FavoritedAt   time.Time
}

type VideoHistoryItem struct {
	VideoID        string
	Title          string
	CoverImageURL  *string
	DurationMS     int32
	ViewCount      int64
	LastPositionMS int32
	LastWatchedAt  time.Time
}

type VideoLibraryPage struct {
	Limit      int
	HasMore    bool
	NextCursor *string
}

type VideoLibraryCursor struct {
	Kind    string
	SortAt  time.Time
	VideoID string
}
