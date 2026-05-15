package model

import (
	"time"
)

type NormalizedLearningEvent struct {
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
}

type NormalizationResult struct {
	Event      *NormalizedLearningEvent
	Skipped    bool
	SkipReason string
}

func Normalized(event NormalizedLearningEvent) NormalizationResult {
	return NormalizationResult{Event: &event}
}

func Skipped(reason string) NormalizationResult {
	return NormalizationResult{Skipped: true, SkipReason: reason}
}
