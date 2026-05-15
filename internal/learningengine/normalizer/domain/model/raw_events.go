package model

import "time"

type RawQuizEvent struct {
	EventID             string
	UserID              string
	QuestionID          string
	CoarseUnitID        int64
	VideoID             string
	RecommendationRunID string
	TriggerType         string
	SelectedOptionIDs   []string
	SelectionIntervalMS []int32
	IsFirstTryCorrect   bool
	TotalElapsedMS      int32
	ShownAt             time.Time
	CompletedAt         time.Time
}

type RawLearningInteraction struct {
	EventID                        string
	UserID                         string
	EventType                      string
	SourceSurface                  string
	VideoID                        string
	WatchSessionID                 string
	RecommendationRunID            string
	RelatedQuizEventID             string
	CoarseUnitID                   int64
	TokenText                      string
	SentenceIndex                  *int32
	SpanIndex                      *int32
	OccurredAt                     time.Time
	ExposureStartMS                *int32
	ExposureEndMS                  *int32
	ExposureCount                  *int32
	LookupVisibleMS                *int32
	LookupSentenceAudioReplayCount int32
	LookupWordAudioPlayCount       int32
	LookupPracticeNowClicked       bool
	EventPayload                   []byte
}
