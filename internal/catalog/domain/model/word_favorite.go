package model

import "time"

type WordFavoriteCoarseKey struct {
	UserID       string
	CoarseUnitID int64
}

type WordFavoriteTokenKey struct {
	UserID        string
	VideoID       string
	SentenceIndex int32
	TokenIndex    int32
}

type WordFavoriteVideoContextKey struct {
	VideoID       string
	SentenceIndex int32
}

type WordFavoriteVideoContext struct {
	VideoTitle          string
	VideoDurationMS     int32
	SentenceText        string
	SentenceTranslation *string
	SentenceStartMS     int32
	SentenceEndMS       int32
}

type SetCoarseWordFavoriteCommand struct {
	UserID        string
	CoarseUnitID  int64
	Source        string
	VideoID       *string
	SentenceIndex *int32
	TokenIndex    *int32
	OccurredAt    time.Time
}

type SetTokenWordFavoriteCommand struct {
	UserID        string
	VideoID       string
	SentenceIndex int32
	TokenIndex    int32
	OccurredAt    time.Time
}

type UnsetCoarseWordFavoriteCommand struct {
	UserID        string
	CoarseUnitID  int64
	Source        string
	VideoID       *string
	SentenceIndex *int32
	TokenIndex    *int32
	OccurredAt    time.Time
}

type UnsetTokenWordFavoriteCommand struct {
	UserID        string
	VideoID       string
	SentenceIndex int32
	TokenIndex    int32
	OccurredAt    time.Time
}

type WordFavoriteListItem struct {
	FavoriteID        string
	FavoritedAt       time.Time
	CoarseUnitID      *int64
	Label             *string
	Pos               *string
	ChineseLabel      *string
	ChineseDef        *string
	Source            string
	VideoID           *string
	SentenceIndex     *int32
	TokenIndex        *int32
	SourceText        *string
	SourceTranslation *string
	SourceDictionary  *string
	SourceExplanation *string
}
