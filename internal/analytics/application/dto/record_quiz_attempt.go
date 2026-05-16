package dto

import "time"

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
	Accepted    bool
	QuizEventID string
	Inserted    bool
}
