package model

type FeedVideoDisplay struct {
	VideoID       string
	Title         string
	CoverImageURL *string
	ViewCount     int64
}

type VideoDetail struct {
	VideoID              string
	Title                string
	Description          string
	VideoObjectPath      string
	CoverImageURL        *string
	TranscriptObjectPath *string
	DurationMS           int32
	ViewCount            int64
	LikeCount            int64
	FavoriteCount        int64
	HasLiked             bool
	HasFavorited         bool
}

type UnitLabel struct {
	CoarseUnitID int64
	Text         string
}
