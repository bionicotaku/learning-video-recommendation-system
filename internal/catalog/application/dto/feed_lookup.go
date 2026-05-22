package dto

type FeedVideoLookupRequest struct {
	UserID   string
	VideoIDs []string
}

type FeedVideoDisplay struct {
	VideoID       string
	Title         string
	CoverImageURL *string
	ViewCount     int64
}

type FeedVideoLookupResponse struct {
	Videos []FeedVideoDisplay
}

type GetVideoDetailRequest struct {
	UserID  string
	VideoID string
}

type VideoDetailResponse struct {
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

type UnitLabelLookupRequest struct {
	CoarseUnitIDs []int64
}

type UnitLabel struct {
	CoarseUnitID int64
	Text         string
}

type UnitLabelLookupResponse struct {
	Labels []UnitLabel
}
