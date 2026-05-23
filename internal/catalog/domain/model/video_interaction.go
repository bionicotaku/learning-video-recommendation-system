package model

import "time"

type VideoLikeCommand struct {
	UserID     string
	VideoID    string
	Enabled    bool
	OccurredAt time.Time
}

type VideoLikeResult struct {
	VideoID   string
	HasLiked  bool
	LikeCount int64
}

type VideoFavoriteCommand struct {
	UserID     string
	VideoID    string
	Enabled    bool
	OccurredAt time.Time
}

type VideoFavoriteResult struct {
	VideoID       string
	HasFavorited  bool
	FavoriteCount int64
}
