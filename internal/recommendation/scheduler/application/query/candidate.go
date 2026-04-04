package query

import "learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"

// ReviewCandidate is a due review candidate returned by the scheduler query layer.
type ReviewCandidate struct {
	State model.UserUnitState
	Unit  model.LearningUnitRef
}

// NewCandidate is a new-learning candidate returned by the scheduler query layer.
type NewCandidate struct {
	State model.UserUnitState
	Unit  model.LearningUnitRef
}
