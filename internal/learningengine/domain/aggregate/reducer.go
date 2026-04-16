package aggregate

import (
	"errors"
	"time"

	"learning-video-recommendation-system/internal/learningengine/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	"learning-video-recommendation-system/internal/learningengine/domain/policy"
)

var ErrLateStrongEvent = errors.New("late strong event")

func Reduce(currentState *model.UserUnitState, event model.LearningEvent) (*model.UserUnitState, error) {
	if err := policy.ValidateEvent(event); err != nil {
		return nil, err
	}

	state := initState(currentState, event)

	if policy.IsStrongEventType(event.EventType) && state.LastReviewedAt != nil && event.OccurredAt.Before(*state.LastReviewedAt) {
		return nil, ErrLateStrongEvent
	}

	updateSeenFields(state, event.OccurredAt)

	if policy.IsWeakEventType(event.EventType) {
		finalizeState(state)
		return state, nil
	}

	quality := *event.Quality
	pass := policy.IsPassingQuality(quality)

	state.StrongEventCount++
	state.LastReviewedAt = timePointer(event.OccurredAt)
	state.LastQuality = int16Pointer(quality)

	if event.EventType == enum.EventReview || event.EventType == enum.EventQuiz {
		state.ReviewCount++
	}

	state.RecentQualityWindow = policy.AppendRecentQuality(state.RecentQualityWindow, quality)
	state.RecentCorrectnessWindow = policy.AppendRecentCorrectness(state.RecentCorrectnessWindow, pass)

	if pass {
		state.CorrectCount++
		state.ConsecutiveCorrect++
		state.ConsecutiveWrong = 0
		policy.ApplySm2Success(state, quality)
	} else {
		state.WrongCount++
		state.ConsecutiveWrong++
		state.ConsecutiveCorrect = 0
		policy.ApplySm2Failure(state)
	}

	state.NextReviewAt = timePointer(event.OccurredAt.Add(time.Duration(state.IntervalDays*24) * time.Hour))

	finalizeState(state)
	return state, nil
}

func RecomputeActiveStatus(state model.UserUnitState) string {
	return policy.ComputeActiveStatus(state)
}

func initState(currentState *model.UserUnitState, event model.LearningEvent) *model.UserUnitState {
	if currentState == nil {
		now := event.OccurredAt
		return &model.UserUnitState{
			UserID:         event.UserID,
			CoarseUnitID:   event.CoarseUnitID,
			IsTarget:       false,
			TargetPriority: 0,
			Status:         enum.StatusNew,
			EaseFactor:     2.5,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
	}

	cloned := *currentState
	cloned.RecentQualityWindow = append([]int16(nil), currentState.RecentQualityWindow...)
	cloned.RecentCorrectnessWindow = append([]bool(nil), currentState.RecentCorrectnessWindow...)
	return &cloned
}

func updateSeenFields(state *model.UserUnitState, occurredAt time.Time) {
	state.SeenCount++
	if state.FirstSeenAt == nil || occurredAt.Before(*state.FirstSeenAt) {
		state.FirstSeenAt = timePointer(occurredAt)
	}
	if state.LastSeenAt == nil || occurredAt.After(*state.LastSeenAt) {
		state.LastSeenAt = timePointer(occurredAt)
	}
}

func finalizeState(state *model.UserUnitState) {
	activeStatus := policy.ComputeActiveStatus(*state)
	state.ProgressPercent = policy.ComputeProgressPercent(*state)
	state.MasteryScore = policy.ComputeMasteryScore(*state)
	state.Status = activeStatus
	if policy.IsSuspendedControl(*state) {
		state.Status = enum.StatusSuspended
	}
	state.UpdatedAt = time.Now().UTC()
}

func timePointer(value time.Time) *time.Time {
	return &value
}

func int16Pointer(value int16) *int16 {
	return &value
}
