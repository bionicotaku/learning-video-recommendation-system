package dto

import "time"

type RecordLearningInteractionsBatchRequest struct {
	UserID              string
	ClientContext       []byte
	VideoID             string
	WatchSessionID      string
	RecommendationRunID string
	Events              []LearningInteractionEvent
}

type LearningInteractionEvent struct {
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
	AcceptedCount  int                                `json:"accepted_count"`
	InsertedCount  int                                `json:"inserted_count"`
	DuplicateCount int                                `json:"duplicate_count"`
	Events         []AcceptedLearningInteractionEvent `json:"events"`
}

type AcceptedLearningInteractionEvent struct {
	ClientEventID              string `json:"client_event_id"`
	LearningInteractionEventID string `json:"learning_interaction_event_id"`
	Inserted                   bool   `json:"inserted"`
}

type RecordQuizAttemptRequest struct {
	UserID        string
	ClientContext []byte

	ClientEventID       string
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

type RecordQuizAttemptResponse struct {
	Accepted    bool   `json:"accepted"`
	QuizEventID string `json:"quiz_event_id"`
	Inserted    bool   `json:"inserted"`
}

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
	Accepted                   bool   `json:"accepted"`
	LearningInteractionEventID string `json:"learning_interaction_event_id"`
	Inserted                   bool   `json:"inserted"`
}
