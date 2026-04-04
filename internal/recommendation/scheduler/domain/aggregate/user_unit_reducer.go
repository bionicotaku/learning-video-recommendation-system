package aggregate

import (
	"fmt"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/policy"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/rule"
	domainservice "learning-video-recommendation-system/internal/recommendation/scheduler/domain/service"
)

const (
	qualityWindowSize     = 5
	correctnessWindowSize = 5
)

type UserUnitReducer interface {
	Reduce(current *model.UserUnitState, event model.LearningEvent, schedulerPolicy policy.SchedulerPolicy) (*model.UserUnitState, error)
}

type userUnitReducer struct {
	weakHandler        rule.WeakEventHandler
	strongHandler      rule.StrongEventHandler
	sm2Updater         domainservice.SM2Updater
	statusTransitioner domainservice.StatusTransitioner
	progressCalculator domainservice.ProgressCalculator
	masteryCalculator  domainservice.MasteryScoreCalculator
}

func NewUserUnitReducer() UserUnitReducer {
	return userUnitReducer{
		weakHandler:        rule.NewWeakEventHandler(),
		strongHandler:      rule.NewStrongEventHandler(),
		sm2Updater:         domainservice.NewSM2Updater(),
		statusTransitioner: domainservice.NewStatusTransitioner(),
		progressCalculator: domainservice.NewProgressCalculator(),
		masteryCalculator:  domainservice.NewMasteryScoreCalculator(),
	}
}

func (r userUnitReducer) Reduce(current *model.UserUnitState, event model.LearningEvent, schedulerPolicy policy.SchedulerPolicy) (*model.UserUnitState, error) {
	if schedulerPolicy.MasteredIntervalDays == 0 {
		schedulerPolicy = policy.DefaultSchedulerPolicy()
	}

	var (
		next *model.UserUnitState
		err  error
	)

	switch event.EventType {
	case enum.EventTypeExposure, enum.EventTypeLookup:
		next, err = r.weakHandler.Apply(current, event)
	case enum.EventTypeNewLearn, enum.EventTypeReview, enum.EventTypeQuiz:
		next, err = r.strongHandler.Apply(current, event)
	default:
		return nil, fmt.Errorf("unsupported event type %q", event.EventType)
	}
	if err != nil {
		return nil, err
	}

	if next.CreatedAt.IsZero() {
		next.CreatedAt = nonZeroTime(event.CreatedAt, event.OccurredAt)
	}
	next.UpdatedAt = nonZeroTime(event.CreatedAt, event.OccurredAt)

	if !isStrongEvent(event.EventType) {
		return next, nil
	}

	if event.Quality != nil {
		next.RecentQualityWindow = appendIntWindow(next.RecentQualityWindow, *event.Quality, qualityWindowSize)
		if err := r.sm2Updater.Apply(next, *event.Quality, event.OccurredAt, schedulerPolicy); err != nil {
			return nil, err
		}
	}
	if event.IsCorrect != nil {
		next.RecentCorrectnessWindow = appendBoolWindow(next.RecentCorrectnessWindow, *event.IsCorrect, correctnessWindowSize)
	}

	if err := r.statusTransitioner.Recompute(next, next.RecentQualityWindow, schedulerPolicy); err != nil {
		return nil, err
	}

	next.ProgressPercent = r.progressCalculator.Compute(next.IntervalDays, schedulerPolicy)
	next.MasteryScore = r.masteryCalculator.Compute(next, recentAccuracy(next.RecentCorrectnessWindow), schedulerPolicy)

	return next, nil
}

func isStrongEvent(eventType enum.EventType) bool {
	switch eventType {
	case enum.EventTypeNewLearn, enum.EventTypeReview, enum.EventTypeQuiz:
		return true
	default:
		return false
	}
}

func nonZeroTime(value, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}

	return value
}

func appendIntWindow(values []int, value, limit int) []int {
	out := append(append([]int(nil), values...), value)
	if len(out) <= limit {
		return out
	}

	return out[len(out)-limit:]
}

func appendBoolWindow(values []bool, value bool, limit int) []bool {
	out := append(append([]bool(nil), values...), value)
	if len(out) <= limit {
		return out
	}

	return out[len(out)-limit:]
}

func recentAccuracy(values []bool) float64 {
	if len(values) == 0 {
		return 0
	}

	correct := 0
	for _, value := range values {
		if value {
			correct++
		}
	}

	return float64(correct) / float64(len(values))
}
