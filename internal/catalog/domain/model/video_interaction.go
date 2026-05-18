package model

type VideoLikeCommand struct {
	UserID  string
	VideoID string
	Enabled bool
}

type VideoLikeResult struct {
	VideoID   string
	HasLiked  bool
	LikeCount int64
}

type VideoFavoriteCommand struct {
	UserID  string
	VideoID string
	Enabled bool
}

type VideoFavoriteResult struct {
	VideoID       string
	HasFavorited  bool
	FavoriteCount int64
}
