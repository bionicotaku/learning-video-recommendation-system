package model

import "time"

type LearningEvent struct {
	EventID         string
	UserID          string
	CoarseUnitID    int64
	VideoID         string
	EventType       string
	ReducerEffect   string
	SourceType      string
	SourceRefID     string
	IsCorrect       *bool
	ProgressQuality *int16
	Metadata        []byte
	OccurredAt      time.Time
	CreatedAt       time.Time
}
