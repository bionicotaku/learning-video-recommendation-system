// 作用：验证 ProgressCalculator 和 MasteryScoreCalculator 的数值公式、关键点和边界截断行为。
// 输入/输出：输入是 intervalDays、UserUnitState、recentAccuracy 和默认策略；输出是数值断言结果。
// 谁调用它：go test、make check。
// 它调用谁/传给谁：调用 domain/service/progress_calculator.go 和 mastery_calculator.go；断言只返回给测试框架。
package service_test

import (
	"math"
	"strconv"
	"testing"

	"learning-video-recommendation-system/internal/learningengine/domain/model"
	"learning-video-recommendation-system/internal/learningengine/domain/policy"
	servicepkg "learning-video-recommendation-system/internal/learningengine/domain/service"
)

func TestProgressCalculatorKeyIntervals(t *testing.T) {
	calculator := servicepkg.NewProgressCalculator()
	schedulerPolicy := policy.DefaultLearningPolicy()

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
	calculator := servicepkg.NewProgressCalculator()
	schedulerPolicy := policy.DefaultLearningPolicy()

	got := calculator.Compute(50, schedulerPolicy)
	if got != 100 {
		t.Fatalf("Compute(50) = %v, want 100", got)
	}
}

func TestMasteryScoreCalculator(t *testing.T) {
	calculator := servicepkg.NewMasteryScoreCalculator()
	schedulerPolicy := policy.DefaultLearningPolicy()

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
	calculator := servicepkg.NewMasteryScoreCalculator()
	schedulerPolicy := policy.DefaultLearningPolicy()

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
