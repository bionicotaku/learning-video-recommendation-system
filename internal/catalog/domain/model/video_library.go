package model

import "time"

type VideoFavoriteListItem struct {
	VideoID       string
	Title         string
	CoverImageURL *string
	DurationMS    int32
	ViewCount     int64
	FavoritedAt   time.Time
}

type VideoHistoryListItem struct {
	VideoID        string
	Title          string
	CoverImageURL  *string
	DurationMS     int32
	ViewCount      int64
	LastPositionMS int32
	LastWatchedAt  time.Time
}
