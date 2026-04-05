package query

import "learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"

// ReviewCandidate is a due review candidate returned by the recommendation query layer.
type ReviewCandidate struct {
	State   model.LearningStateSnapshot
	Unit    model.CoarseUnitRef
	Serving model.UserUnitServingState
}

// NewCandidate is a new-learning candidate returned by the recommendation query layer.
type NewCandidate struct {
	State   model.LearningStateSnapshot
	Unit    model.CoarseUnitRef
	Serving model.UserUnitServingState
}

// ScoredReviewCandidate is a review candidate with its computed score and reasons.
type ScoredReviewCandidate struct {
	Candidate   ReviewCandidate
	Score       float64
	ReasonCodes []string
}

// ScoredNewCandidate is a new candidate with its computed score and reasons.
type ScoredNewCandidate struct {
	Candidate   NewCandidate
	Score       float64
	ReasonCodes []string
}
