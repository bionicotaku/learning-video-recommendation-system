package aggregate

import (
	"errors"
	"time"

	"learning-video-recommendation-system/internal/learningengine/reducer/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/policy"
)

var ErrLateProgressEvent = errors.New("late progress event")

func Reduce(currentState *model.UserUnitState, event model.LearningEvent) (*model.UserUnitState, error) {
	if err := policy.ValidateEvent(event); err != nil {
		return nil, err
	}

	state := initState(currentState, event)

	if policy.IsResetUnlearnedEffect(event.ReducerEffect) {
		applyResetUnlearnedState(state)
		state.UpdatedAt = time.Now().UTC()
		return state, nil
	}

	if isTerminalMastered(state) && !policy.IsSetMasteredEffect(event.ReducerEffect) {
		return state, nil
	}

	if policy.IsAffectsProgressEffect(event.ReducerEffect) && state.LastProgressAt != nil && event.OccurredAt.Before(*state.LastProgressAt) {
		return nil, ErrLateProgressEvent
	}

	updateObservationFields(state, event.OccurredAt)

	if policy.IsSetMasteredEffect(event.ReducerEffect) {
		state.LastProgressAt = timePointer(event.OccurredAt)
		applyCompletedMasteredState(state)
		state.UpdatedAt = time.Now().UTC()
		return state, nil
	}

	if policy.IsObserveOnlyEffect(event.ReducerEffect) {
		finalizeState(state)
		return state, nil
	}

	quality := *event.ProgressQuality
	pass := policy.IsPassingQuality(quality)

	state.ProgressEventCount++
	state.LastProgressAt = timePointer(event.OccurredAt)
	state.LastProgressQuality = int16Pointer(quality)
	state.RecentProgressQualities = policy.AppendRecentProgressQuality(state.RecentProgressQualities, quality)
	state.RecentProgressPasses = policy.AppendRecentProgressPass(state.RecentProgressPasses, pass)

	if pass {
		state.ProgressSuccessCount++
		if event.CountsTowardSuccessStreak {
			state.ConsecutiveSuccessCount++
			state.ConsecutiveFailureCount = 0
		}
		policy.ApplyProgressSuccessSchedule(state, quality)
	} else {
		state.ProgressFailureCount++
		if event.CountsTowardSuccessStreak {
			state.ConsecutiveFailureCount++
			state.ConsecutiveSuccessCount = 0
		}
		policy.ApplyProgressFailureSchedule(state)
	}

	state.NextReviewAt = timePointer(event.OccurredAt.Add(time.Duration(state.ScheduleIntervalDays*24) * time.Hour))

	finalizeState(state)
	return state, nil
}

func initState(currentState *model.UserUnitState, event model.LearningEvent) *model.UserUnitState {
	if currentState == nil {
		now := event.OccurredAt
		return &model.UserUnitState{
			UserID:             event.UserID,
			CoarseUnitID:       event.CoarseUnitID,
			IsTarget:           false,
			TargetPriority:     0,
			Status:             enum.StatusNew,
			ScheduleEaseFactor: 2.5,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
	}

	cloned := *currentState
	cloned.RecentProgressQualities = append([]int16(nil), currentState.RecentProgressQualities...)
	cloned.RecentProgressPasses = append([]bool(nil), currentState.RecentProgressPasses...)
	if cloned.ScheduleEaseFactor == 0 {
		cloned.ScheduleEaseFactor = 2.5
	}
	return &cloned
}

func updateObservationFields(state *model.UserUnitState, occurredAt time.Time) {
	state.ObservationCount++
	if state.FirstObservedAt == nil || occurredAt.Before(*state.FirstObservedAt) {
		state.FirstObservedAt = timePointer(occurredAt)
	}
	if state.LastObservedAt == nil || occurredAt.After(*state.LastObservedAt) {
		state.LastObservedAt = timePointer(occurredAt)
	}
}

func finalizeState(state *model.UserUnitState) {
	state.ProgressPercent = policy.ComputeProgressPercent(*state)
	state.MasteryScore = policy.ComputeMasteryScore(*state)
	if state.ProgressPercent >= 100 {
		applyCompletedMasteredState(state)
		state.UpdatedAt = time.Now().UTC()
		return
	}
	activeStatus := policy.ComputeActiveStatus(*state)
	state.Status = activeStatus
	if activeStatus == enum.StatusMastered {
		applyCompletedMasteredState(state)
		state.UpdatedAt = time.Now().UTC()
		return
	}
	state.UpdatedAt = time.Now().UTC()
}

func isTerminalMastered(state *model.UserUnitState) bool {
	return state.Status == enum.StatusMastered
}

func applyCompletedMasteredState(state *model.UserUnitState) {
	state.Status = enum.StatusMastered
	state.ProgressPercent = 100
	state.MasteryScore = 1
	state.NextReviewAt = nil
}

func applyResetUnlearnedState(state *model.UserUnitState) {
	state.Status = enum.StatusNew
	state.ProgressPercent = 0
	state.MasteryScore = 0
	state.FirstObservedAt = nil
	state.LastObservedAt = nil
	state.ObservationCount = 0
	state.ProgressEventCount = 0
	state.LastProgressAt = nil
	state.LastProgressQuality = nil
	state.RecentProgressQualities = nil
	state.RecentProgressPasses = nil
	state.ProgressSuccessCount = 0
	state.ProgressFailureCount = 0
	state.ConsecutiveSuccessCount = 0
	state.ConsecutiveFailureCount = 0
	state.ScheduleRepetition = 0
	state.ScheduleIntervalDays = 0
	state.ScheduleEaseFactor = 2.5
	state.NextReviewAt = nil
}

func timePointer(value time.Time) *time.Time {
	return &value
}

func int16Pointer(value int16) *int16 {
	return &value
}
