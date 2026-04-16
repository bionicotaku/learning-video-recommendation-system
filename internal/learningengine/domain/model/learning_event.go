package model

import "time"

type LearningEvent struct {
	EventID        int64
	UserID         string
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
	CreatedAt      time.Time
}
