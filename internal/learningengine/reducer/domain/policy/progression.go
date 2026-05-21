package policy

import (
	"math"

	"learning-video-recommendation-system/internal/learningengine/reducer/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
)

const (
	recentWindowLimit             = 5
	easeFactorFloor               = 1.3
	firstReviewIntervalDays       = 0.25
	targetCompletionIntervalDays  = 4.0
	maxScheduleIntervalDays       = 14.0
	targetCompletionSuccessStreak = 3
	targetCompletionMasteryScore  = 0.75
)

func RecentWindowLimit() int {
	return recentWindowLimit
}

func ApplyProgressSuccessSchedule(state *model.UserUnitState, quality int16) {
	state.ScheduleRepetition++

	switch state.ScheduleRepetition {
	case 1:
		state.ScheduleIntervalDays = firstReviewIntervalDays
	case 2:
		state.ScheduleIntervalDays = 1
	case 3:
		state.ScheduleIntervalDays = 2
	case 4:
		state.ScheduleIntervalDays = targetCompletionIntervalDays
	default:
		state.ScheduleIntervalDays = math.Min(maxScheduleIntervalDays, math.Round(state.ScheduleIntervalDays*state.ScheduleEaseFactor))
	}

	delta := 0.1 - float64(5-quality)*(0.08+float64(5-quality)*0.02)
	state.ScheduleEaseFactor = math.Max(easeFactorFloor, state.ScheduleEaseFactor+delta)
}

func ApplyProgressFailureSchedule(state *model.UserUnitState) {
	state.ScheduleRepetition = 0
	state.ScheduleIntervalDays = firstReviewIntervalDays
}

func ComputeProgressPercent(state model.UserUnitState) float64 {
	if state.ProgressEventCount == 0 {
		return 0
	}
	if ComputeActiveStatus(state) == enum.StatusMastered {
		return 100
	}

	intervalComponent := minFloat(state.ScheduleIntervalDays/targetCompletionIntervalDays, 1)
	accuracyComponent := averageBoolWindow(state.RecentProgressPasses)
	repetitionComponent := minFloat(float64(state.ScheduleRepetition)/targetCompletionSuccessStreak, 1)
	qualityComponent := averageInt16Window(state.RecentProgressQualities) / 5

	score := 100 * clamp01(
		0.45*intervalComponent+
			0.20*accuracyComponent+
			0.20*repetitionComponent+
			0.15*qualityComponent,
	)

	return round(score, 2)
}

func ComputeMasteryScore(state model.UserUnitState) float64 {
	if state.ProgressEventCount == 0 {
		return 0
	}

	intervalComponent := minFloat(state.ScheduleIntervalDays/targetCompletionIntervalDays, 1)
	accuracyComponent := averageBoolWindow(state.RecentProgressPasses)
	stabilityComponent := minFloat(float64(state.ConsecutiveSuccessCount)/targetCompletionSuccessStreak, 1)
	repetitionComponent := minFloat(float64(state.ScheduleRepetition)/targetCompletionSuccessStreak, 1)

	failurePenalty := 0.0
	if state.LastProgressQuality != nil && *state.LastProgressQuality < 3 {
		failurePenalty = 0.20
	}

	score := clamp01(
		0.45*intervalComponent +
			0.25*accuracyComponent +
			0.15*stabilityComponent +
			0.15*repetitionComponent -
			failurePenalty,
	)

	return round(score, 4)
}

func ComputeActiveStatus(state model.UserUnitState) string {
	if state.ProgressEventCount == 0 {
		return enum.StatusNew
	}

	if state.LastProgressQuality != nil && *state.LastProgressQuality < 3 {
		return enum.StatusReviewing
	}

	if !hasTwoRecentPassingQualities(state.RecentProgressQualities) || state.ProgressEventCount < 2 || state.ScheduleIntervalDays < 1 {
		return enum.StatusLearning
	}

	if state.ConsecutiveSuccessCount >= targetCompletionSuccessStreak && noRecentFailures(state.RecentProgressPasses) && ComputeMasteryScore(state) >= targetCompletionMasteryScore {
		return enum.StatusMastered
	}

	return enum.StatusReviewing
}

func IsSuspendedControl(state model.UserUnitState) bool {
	return state.Status == enum.StatusSuspended || state.SuspendedReason != ""
}

func AppendRecentProgressQuality(values []int16, quality int16) []int16 {
	return trimRecentWindow(append(values, quality), recentWindowLimit)
}

func AppendRecentProgressPass(values []bool, pass bool) []bool {
	return trimRecentWindow(append(values, pass), recentWindowLimit)
}

func trimRecentWindow[T any](values []T, limit int) []T {
	if len(values) <= limit {
		return values
	}
	trimmed := make([]T, limit)
	copy(trimmed, values[len(values)-limit:])
	return trimmed
}

func hasTwoRecentPassingQualities(values []int16) bool {
	if len(values) < 2 {
		return false
	}
	last := values[len(values)-2:]
	return last[0] >= 3 && last[1] >= 3
}

func noRecentFailures(values []bool) bool {
	if len(values) == 0 {
		return false
	}

	start := 0
	if len(values) > 3 {
		start = len(values) - 3
	}

	for _, value := range values[start:] {
		if !value {
			return false
		}
	}
	return true
}

func averageBoolWindow(values []bool) float64 {
	if len(values) == 0 {
		return 0
	}

	total := 0.0
	for _, value := range values {
		if value {
			total++
		}
	}
	return total / float64(len(values))
}

func averageInt16Window(values []int16) float64 {
	if len(values) == 0 {
		return 0
	}

	total := 0.0
	for _, value := range values {
		total += float64(value)
	}
	return total / float64(len(values))
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func minFloat(value float64, max float64) float64 {
	if value > max {
		return max
	}
	return value
}

func round(value float64, decimals int) float64 {
	scale := math.Pow10(decimals)
	return math.Round(value*scale) / scale
}
