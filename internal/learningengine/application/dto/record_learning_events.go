package dto

import "time"

type LearningEventInput struct {
	CoarseUnitID   int64
	VideoID        string
	EventType      string
	SourceType     string
	SourceRefID    string
	IsCorrect      *bool
	Quality        *int16
	ResponseTimeMs *int32
	Metadata       []byte
	OccurredAt     time.Time
}

type RecordLearningEventsRequest struct {
	UserID string
	Events []LearningEventInput
}

type RecordLearningEventsResponse struct {
	RecordedCount int
}
