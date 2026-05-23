package dto

import "time"

type ResetUserUnitProgressRequest struct {
	UserID        string
	ClientEventID string
	CoarseUnitID  int64
	SourceSurface string
	VideoID       string

	WatchSessionID      string
	RecommendationRunID string
	RelatedQuizEventID  string
	TokenText           string
	SentenceIndex       *int32
	SpanIndex           *int32
	OccurredAt          time.Time
	ClientContext       []byte
	EventPayload        []byte
}

type ResetUserUnitProgressResponse struct {
	Accepted            bool
	UnitLearningEventID string
	Inserted            bool
}
