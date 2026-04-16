package policy

import (
	"math"

	"learning-video-recommendation-system/internal/learningengine/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
)

const (
	recentWindowLimit = 5
	easeFactorFloor   = 1.3
)

func RecentWindowLimit() int {
	return recentWindowLimit
}

func ApplySm2Success(state *model.UserUnitState, quality int16) {
	state.Repetition++

	switch state.Repetition {
	case 1:
		state.IntervalDays = 1
	case 2:
		state.IntervalDays = 3
	case 3:
		state.IntervalDays = 6
	default:
		state.IntervalDays = math.Round(state.IntervalDays * state.EaseFactor)
	}

	delta := 0.1 - float64(5-quality)*(0.08+float64(5-quality)*0.02)
	state.EaseFactor = math.Max(easeFactorFloor, state.EaseFactor+delta)
}

func ApplySm2Failure(state *model.UserUnitState) {
	state.Repetition = 0
	state.IntervalDays = 1
}

func ComputeProgressPercent(state model.UserUnitState) float64 {
	if state.StrongEventCount == 0 {
		return 0
	}
	if ComputeActiveStatus(state) == enum.StatusMastered {
		return 100
	}

	intervalComponent := minFloat(state.IntervalDays/21, 1)
	accuracyComponent := averageBoolWindow(state.RecentCorrectnessWindow)
	repetitionComponent := minFloat(float64(state.Repetition)/4, 1)
	qualityComponent := averageInt16Window(state.RecentQualityWindow) / 5

	score := 100 * clamp01(
		0.45*intervalComponent+
			0.20*accuracyComponent+
			0.20*repetitionComponent+
			0.15*qualityComponent,
	)

	return round(score, 2)
}

func ComputeMasteryScore(state model.UserUnitState) float64 {
	if state.StrongEventCount == 0 {
		return 0
	}

	intervalComponent := minFloat(state.IntervalDays/21, 1)
	accuracyComponent := averageBoolWindow(state.RecentCorrectnessWindow)
	stabilityComponent := minFloat(float64(state.ConsecutiveCorrect)/3, 1)
	repetitionComponent := minFloat(float64(state.Repetition)/4, 1)

	failurePenalty := 0.0
	if state.LastQuality != nil && *state.LastQuality < 3 {
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
	if state.StrongEventCount == 0 {
		return enum.StatusNew
	}

	if state.LastQuality != nil && *state.LastQuality < 3 {
		return enum.StatusReviewing
	}

	if !hasTwoRecentPassingQualities(state.RecentQualityWindow) || state.StrongEventCount < 2 || state.IntervalDays < 3 {
		return enum.StatusLearning
	}

	if state.IntervalDays >= 21 && noRecentFailures(state.RecentCorrectnessWindow) && ComputeMasteryScore(state) >= 0.8 {
		return enum.StatusMastered
	}

	return enum.StatusReviewing
}

func IsSuspendedControl(state model.UserUnitState) bool {
	return state.Status == enum.StatusSuspended || state.SuspendedReason != ""
}

func AppendRecentQuality(values []int16, quality int16) []int16 {
	return trimRecentWindow(append(values, quality), recentWindowLimit)
}

func AppendRecentCorrectness(values []bool, correct bool) []bool {
	return trimRecentWindow(append(values, correct), recentWindowLimit)
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
