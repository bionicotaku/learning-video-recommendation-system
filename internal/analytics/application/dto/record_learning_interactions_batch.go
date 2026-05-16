package dto

import "time"

type RecordLearningInteractionsBatchRequest struct {
	UserID              string
	ClientContext       []byte
	VideoID             string
	WatchSessionID      string
	RecommendationRunID string
	Events              []LearningInteractionEventInput
}

type LearningInteractionEventInput struct {
	ClientEventID string

	EventType     string
	SourceSurface string
	CoarseUnitID  *int64
	TokenText     string
	SentenceIndex *int32
	SpanIndex     *int32
	OccurredAt    time.Time

	ExposureStartMS *int32
	ExposureEndMS   *int32
	ExposureCount   *int32

	LookupVisibleMS                *int32
	LookupSentenceAudioReplayCount int32
	LookupWordAudioPlayCount       int32
	LookupPracticeNowClicked       bool

	EventPayload []byte
}

type RecordLearningInteractionsBatchResponse struct {
	AcceptedCount  int
	InsertedCount  int
	DuplicateCount int
	AcceptedEvents []AcceptedLearningInteractionEvent
}

type AcceptedLearningInteractionEvent struct {
	ClientEventID              string
	LearningInteractionEventID string
	Inserted                   bool
}
