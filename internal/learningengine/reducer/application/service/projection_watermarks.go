package service

import (
	"time"

	"learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
)

func applyLearningEventProjection(state *model.UserUnitState, event model.LearningEvent) {
	if state == nil {
		return
	}
	if state.LatestLearningEventOccurredAt == nil || event.OccurredAt.After(*state.LatestLearningEventOccurredAt) {
		occurredAt := event.OccurredAt.UTC()
		state.LatestLearningEventOccurredAt = &occurredAt
	}
	if event.ResetBoundaryAt != nil && (state.LatestResetBoundaryAt == nil || event.ResetBoundaryAt.After(*state.LatestResetBoundaryAt)) {
		resetBoundaryAt := event.ResetBoundaryAt.UTC()
		state.LatestResetBoundaryAt = &resetBoundaryAt
	}
	if event.LedgerSeq > state.LatestLearningEventLedgerSeq {
		state.LatestLearningEventLedgerSeq = event.LedgerSeq
	}
}

func resetBoundaryFromState(clientOccurredAt time.Time, state *model.UserUnitState) time.Time {
	boundary := clientOccurredAt.UTC()
	if state == nil {
		return boundary
	}
	if state.LatestLearningEventOccurredAt != nil && state.LatestLearningEventOccurredAt.After(boundary) {
		boundary = state.LatestLearningEventOccurredAt.UTC()
	}
	if state.LatestResetBoundaryAt != nil && state.LatestResetBoundaryAt.After(boundary) {
		boundary = state.LatestResetBoundaryAt.UTC()
	}
	return boundary
}
