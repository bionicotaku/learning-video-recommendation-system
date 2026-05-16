package dto

import "time"

const (
	SourceKindAll                 = "all"
	SourceKindQuiz                = "quiz"
	SourceKindLearningInteraction = "learning_interaction"
	DefaultNormalizeLimit         = 500
	MaxNormalizeLimit             = 1000
)

type NormalizePendingEventsRequest struct {
	UserID         string
	SourceKind     string
	Limit          int
	OccurredBefore *time.Time
}

type NormalizePendingEventsResponse struct {
	ReadRawCount           int
	NormalizedEventCount   int
	SkippedCount           int
	RecordedEventCount     int
	DuplicateEventCount    int
	ErrorCount             int
	RecordedUserBatchCount int
}

type NormalizeLearningInteractionsByIDsRequest struct {
	UserID                      string
	LearningInteractionEventIDs []string
}

type NormalizeLearningInteractionsByIDsResponse struct {
	ReadRawCount           int
	NormalizedEventCount   int
	SkippedCount           int
	RecordedEventCount     int
	DuplicateEventCount    int
	ErrorCount             int
	RecordedUserBatchCount int
}

type NormalizeQuizAttemptByIDRequest struct {
	UserID      string
	QuizEventID string
}

type NormalizeQuizAttemptByIDResponse struct {
	ReadRawCount           int
	NormalizedEventCount   int
	SkippedCount           int
	RecordedEventCount     int
	DuplicateEventCount    int
	ErrorCount             int
	RecordedUserBatchCount int
}
