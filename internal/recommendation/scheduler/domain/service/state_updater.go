package service

import (
	"fmt"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/policy"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/rule"
)

type UpdateContext struct {
	SchedulerPolicy   policy.SchedulerPolicy
	RecentQualities   []int
	RecentCorrectness []bool
	Now               time.Time
}

type StateTransitionResult struct {
	PreviousStatus      enum.UnitStatus
	NewStatus           enum.UnitStatus
	StatusChanged       bool
	ProgressChanged     bool
	MasteryScoreChanged bool
	NextReviewChanged   bool
}

type StateUpdater interface {
	Apply(current *model.UserUnitState, event model.LearningEvent, ctx UpdateContext) (*model.UserUnitState, StateTransitionResult, error)
}

type stateUpdater struct {
	weakHandler        rule.WeakEventHandler
	strongHandler      rule.StrongEventHandler
	sm2Updater         SM2Updater
	statusTransitioner StatusTransitioner
	progressCalculator ProgressCalculator
	masteryCalculator  MasteryScoreCalculator
}

func NewStateUpdater() StateUpdater {
	return stateUpdater{
		weakHandler:        rule.NewWeakEventHandler(),
		strongHandler:      rule.NewStrongEventHandler(),
		sm2Updater:         NewSM2Updater(),
		statusTransitioner: NewStatusTransitioner(),
		progressCalculator: NewProgressCalculator(),
		masteryCalculator:  NewMasteryScoreCalculator(),
	}
}

func (u stateUpdater) Apply(current *model.UserUnitState, event model.LearningEvent, ctx UpdateContext) (*model.UserUnitState, StateTransitionResult, error) {
	if ctx.SchedulerPolicy.MasteredIntervalDays == 0 {
		ctx.SchedulerPolicy = policy.DefaultSchedulerPolicy()
	}

	var (
		next *model.UserUnitState
		err  error
	)

	switch event.EventType {
	case enum.EventTypeExposure, enum.EventTypeLookup:
		next, err = u.weakHandler.Apply(current, event)
	case enum.EventTypeNewLearn, enum.EventTypeReview, enum.EventTypeQuiz:
		next, err = u.strongHandler.Apply(current, event)
	default:
		return nil, StateTransitionResult{}, fmt.Errorf("unsupported event type %q", event.EventType)
	}
	if err != nil {
		return nil, StateTransitionResult{}, err
	}

	previousStatus := next.Status
	previousProgress := next.ProgressPercent
	previousMastery := next.MasteryScore
	previousNextReviewAt := cloneTimePtr(next.NextReviewAt)

	if next.CreatedAt.IsZero() {
		next.CreatedAt = event.OccurredAt
	}
	if ctx.Now.IsZero() {
		next.UpdatedAt = event.OccurredAt
	} else {
		next.UpdatedAt = ctx.Now
	}

	if isStrongEvent(event.EventType) {
		if event.Quality != nil {
			if err := u.sm2Updater.Apply(next, *event.Quality, event.OccurredAt, ctx.SchedulerPolicy); err != nil {
				return nil, StateTransitionResult{}, err
			}
		}

		recentQualities := appendRecentQuality(ctx.RecentQualities, event.Quality)
		if err := u.statusTransitioner.Recompute(next, recentQualities, ctx.SchedulerPolicy); err != nil {
			return nil, StateTransitionResult{}, err
		}

		next.ProgressPercent = u.progressCalculator.Compute(next.IntervalDays, ctx.SchedulerPolicy)
		next.MasteryScore = u.masteryCalculator.Compute(next, recentAccuracy(appendRecentCorrectness(ctx.RecentCorrectness, event.IsCorrect)), ctx.SchedulerPolicy)
	}

	return next, StateTransitionResult{
		PreviousStatus:      previousStatus,
		NewStatus:           next.Status,
		StatusChanged:       previousStatus != next.Status,
		ProgressChanged:     previousProgress != next.ProgressPercent,
		MasteryScoreChanged: previousMastery != next.MasteryScore,
		NextReviewChanged:   !sameTimePtr(previousNextReviewAt, next.NextReviewAt),
	}, nil
}

func isStrongEvent(eventType enum.EventType) bool {
	switch eventType {
	case enum.EventTypeNewLearn, enum.EventTypeReview, enum.EventTypeQuiz:
		return true
	default:
		return false
	}
}

func appendRecentQuality(qualities []int, quality *int) []int {
	if quality == nil {
		copied := make([]int, len(qualities))
		copy(copied, qualities)
		return copied
	}

	out := make([]int, 0, len(qualities)+1)
	out = append(out, qualities...)
	out = append(out, *quality)
	return out
}

func appendRecentCorrectness(correctness []bool, isCorrect *bool) []bool {
	if isCorrect == nil {
		copied := make([]bool, len(correctness))
		copy(copied, correctness)
		return copied
	}

	out := make([]bool, 0, len(correctness)+1)
	out = append(out, correctness...)
	out = append(out, *isCorrect)
	return out
}

func recentAccuracy(correctness []bool) float64 {
	if len(correctness) == 0 {
		return 0
	}

	start := 0
	if len(correctness) > 5 {
		start = len(correctness) - 5
	}

	correctCount := 0
	for _, value := range correctness[start:] {
		if value {
			correctCount++
		}
	}

	return float64(correctCount) / float64(len(correctness[start:]))
}

func sameTimePtr(left, right *time.Time) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return left.Equal(*right)
	}
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	copied := *value
	return &copied
}
