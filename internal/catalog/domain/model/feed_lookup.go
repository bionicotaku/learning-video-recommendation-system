package model

type FeedVideoDisplay struct {
	VideoID               string
	Title                 string
	Description           string
	HLSMasterPlaylistPath string
	CoverImageURL         *string
	ViewCount             int64
	LikeCount             int64
	FavoriteCount         int64
}

type UnitLabel struct {
	CoarseUnitID int64
	Text         string
}
