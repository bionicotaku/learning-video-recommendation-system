package model

import "time"

type LearningEvent struct {
	EventID                   string
	LedgerSeq                 int64
	UserID                    string
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
	ResetBoundaryAt           *time.Time
	CreatedAt                 time.Time
}

type UnitLearningEventWatermark struct {
	CoarseUnitID       int64
	MaxOccurredAt      *time.Time
	MaxResetBoundaryAt *time.Time
}
