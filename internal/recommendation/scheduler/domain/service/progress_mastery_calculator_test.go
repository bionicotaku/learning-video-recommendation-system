package service

import (
	"math"
	"strconv"
	"testing"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/policy"
)

func TestProgressCalculatorKeyIntervals(t *testing.T) {
	calculator := NewProgressCalculator()
	schedulerPolicy := policy.DefaultSchedulerPolicy()

	tests := []struct {
		intervalDays float64
		want         float64
	}{
		{intervalDays: 0, want: 0},
		{intervalDays: 1, want: math.Log(2) / math.Log(22) * 100},
		{intervalDays: 3, want: math.Log(4) / math.Log(22) * 100},
		{intervalDays: 6, want: math.Log(7) / math.Log(22) * 100},
		{intervalDays: 21, want: 100},
	}

	for _, tt := range tests {
		t.Run(formatIntervalLabel(tt.intervalDays), func(t *testing.T) {
			got := calculator.Compute(tt.intervalDays, schedulerPolicy)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Fatalf("Compute(%v) = %v, want %v", tt.intervalDays, got, tt.want)
			}
		})
	}
}

func TestProgressCalculatorClampsBeyondMasteredThreshold(t *testing.T) {
	calculator := NewProgressCalculator()
	schedulerPolicy := policy.DefaultSchedulerPolicy()

	got := calculator.Compute(50, schedulerPolicy)
	if got != 100 {
		t.Fatalf("Compute(50) = %v, want 100", got)
	}
}

func TestMasteryScoreCalculator(t *testing.T) {
	calculator := NewMasteryScoreCalculator()
	schedulerPolicy := policy.DefaultSchedulerPolicy()

	state := &model.UserUnitState{
		IntervalDays:    6,
		ProgressPercent: math.Log(7) / math.Log(22) * 100,
	}

	got := calculator.Compute(state, 0.8, schedulerPolicy)
	want := 0.45*(state.ProgressPercent/100) + 0.35*0.8 + 0.20*(6.0/21.0)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("Compute() = %v, want %v", got, want)
	}
}

func TestMasteryScoreCalculatorClampsToUnitInterval(t *testing.T) {
	calculator := NewMasteryScoreCalculator()
	schedulerPolicy := policy.DefaultSchedulerPolicy()

	state := &model.UserUnitState{
		IntervalDays:    30,
		ProgressPercent: 110,
	}

	got := calculator.Compute(state, 2, schedulerPolicy)
	if got != 1 {
		t.Fatalf("Compute() = %v, want 1", got)
	}
}

func formatIntervalLabel(intervalDays float64) string {
	if intervalDays == math.Trunc(intervalDays) {
		return "interval_" + strconv.Itoa(int(intervalDays))
	}

	return "interval_float"
}
