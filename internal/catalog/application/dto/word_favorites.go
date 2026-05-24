package dto

import "time"

const (
	WordFavoriteSourceWordList        = "word_list"
	WordFavoriteSourceVideoTranscript = "video_transcript"

	WordFavoriteKeyTypeCoarseUnit = "coarse_unit"
	WordFavoriteKeyTypeVideoToken = "video_token"

	WordFavoritesCursorKind = "word_favorites"
)

type GetWordFavoriteStatusRequest struct {
	UserID          string
	CoarseUnitID    *int64
	Text            string
	Source          string
	VideoID         *string
	SentenceIndex   *int32
	TokenIndex      *int32
	IncludeVideoCtx bool
}

type SetWordFavoriteRequest struct {
	UserID        string
	CoarseUnitID  *int64
	Text          string
	Source        string
	VideoID       *string
	SentenceIndex *int32
	TokenIndex    *int32
	OccurredAt    time.Time
}

type UnsetWordFavoriteRequest struct {
	UserID        string
	CoarseUnitID  *int64
	Text          string
	Source        string
	VideoID       *string
	SentenceIndex *int32
	TokenIndex    *int32
	OccurredAt    time.Time
}

type ListWordFavoritesRequest struct {
	UserID string
	Limit  int
	Cursor string
}

type ListWordFavoritesQuery struct {
	UserID       string
	LimitPlusOne int
	Cursor       *WordFavoritesCursor
}

type WordFavoriteStatusResponse struct {
	IsFavorited  bool                      `json:"is_favorited"`
	VideoContext *WordFavoriteVideoContext `json:"video_context,omitempty"`
}

type WordFavoriteVideoContext struct {
	VideoID             string  `json:"video_id"`
	VideoTitle          string  `json:"video_title"`
	VideoDurationMS     *int32  `json:"video_duration_ms"`
	TokenIndex          int32   `json:"token_index"`
	SentenceIndex       int32   `json:"sentence_index"`
	SentenceText        string  `json:"sentence_text"`
	SentenceTranslation *string `json:"sentence_translation"`
	SentenceStartMS     *int32  `json:"sentence_start_ms"`
	SentenceEndMS       *int32  `json:"sentence_end_ms"`
}

type WordFavoriteListPage struct {
	Items []WordFavoriteListItem `json:"items"`
	Page  WordFavoritePage       `json:"page"`
}

type WordFavoriteListItem struct {
	CoarseUnitID      *int64  `json:"coarse_unit_id"`
	Label             *string `json:"label"`
	Pos               *string `json:"pos"`
	ChineseLabel      *string `json:"chinese_label"`
	ChineseDef        *string `json:"chinese_def"`
	Source            string  `json:"source"`
	VideoID           *string `json:"video_id"`
	SentenceIndex     *int32  `json:"sentence_index"`
	TokenIndex        *int32  `json:"token_index"`
	SourceText        *string `json:"source_text"`
	SourceTranslation *string `json:"source_translation"`
	SourceDictionary  *string `json:"source_dictionary"`
	SourceExplanation *string `json:"source_explanation"`
}

type WordFavoritePage struct {
	Limit      int     `json:"limit"`
	HasMore    bool    `json:"has_more"`
	NextCursor *string `json:"next_cursor"`
}

type WordFavoritesCursor struct {
	Kind        string
	FavoritedAt time.Time
	FavoriteID  string
}
