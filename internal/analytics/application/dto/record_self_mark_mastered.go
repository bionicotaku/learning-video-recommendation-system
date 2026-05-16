package dto

import "time"

type RecordSelfMarkMasteredRequest struct {
	UserID        string
	ClientContext []byte

	ClientEventID       string
	CoarseUnitID        int64
	SourceSurface       string
	VideoID             string
	WatchSessionID      string
	RecommendationRunID string
	RelatedQuizEventID  string
	TokenText           string
	SentenceIndex       *int32
	SpanIndex           *int32
	OccurredAt          time.Time
	EventPayload        []byte
}

type RecordSelfMarkMasteredResponse struct {
	Accepted                   bool
	LearningInteractionEventID string
	Inserted                   bool
}
