package dto

import "time"

type LearningEventInput struct {
	CoarseUnitID              int64
	VideoID                   string
	EventType                 string
	ReducerEffect             string
	SourceType                string
	SourceRefID               string
	IsCorrect                 *bool
	ProgressQuality           *int16
	CountsTowardSuccessStreak bool
	ConsumedWatchSessionIDs   []string
	Metadata                  []byte
	OccurredAt                time.Time
}

type RecordLearningEventsRequest struct {
	UserID string
	Events []LearningEventInput
}

type RecordLearningEventsResponse struct {
	ReceivedCount  int
	RecordedCount  int
	DuplicateCount int
}
