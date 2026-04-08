// 作用：验证 StatusTransitioner 的生命周期迁移和不满足条件时的禁止升级行为。
// 输入/输出：输入是伪造的 UserUnitState、recent quality window 和默认策略；输出是状态迁移断言结果。
// 谁调用它：go test、make check。
// 它调用谁/传给谁：调用 domain/service/status_transitioner.go；断言只返回给测试框架。
package service_test

import (
	"testing"

	"learning-video-recommendation-system/internal/learningengine/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	"learning-video-recommendation-system/internal/learningengine/domain/policy"
	servicepkg "learning-video-recommendation-system/internal/learningengine/domain/service"
)

func TestStatusTransitionerCoversFullLifecycle(t *testing.T) {
	transitioner := servicepkg.NewStatusTransitioner()
	schedulerPolicy := policy.DefaultLearningPolicy()

	state := &model.UserUnitState{
		Status:           enum.UnitStatusNew,
		StrongEventCount: 1,
	}

	if err := transitioner.Recompute(state, []int{4}, schedulerPolicy); err != nil {
		t.Fatalf("Recompute() new->learning error = %v", err)
	}
	if state.Status != enum.UnitStatusLearning {
		t.Fatalf("Status = %q, want %q", state.Status, enum.UnitStatusLearning)
	}

	state.StrongEventCount = 2
	if err := transitioner.Recompute(state, []int{4, 5}, schedulerPolicy); err != nil {
		t.Fatalf("Recompute() learning->reviewing error = %v", err)
	}
	if state.Status != enum.UnitStatusReviewing {
		t.Fatalf("Status = %q, want %q", state.Status, enum.UnitStatusReviewing)
	}

	state.IntervalDays = schedulerPolicy.MasteredIntervalDays
	state.ConsecutiveWrong = 0
	if err := transitioner.Recompute(state, []int{4, 5}, schedulerPolicy); err != nil {
		t.Fatalf("Recompute() reviewing->mastered error = %v", err)
	}
	if state.Status != enum.UnitStatusMastered {
		t.Fatalf("Status = %q, want %q", state.Status, enum.UnitStatusMastered)
	}

	state.ConsecutiveWrong = 1
	if err := transitioner.Recompute(state, []int{2}, schedulerPolicy); err != nil {
		t.Fatalf("Recompute() mastered->reviewing error = %v", err)
	}
	if state.Status != enum.UnitStatusReviewing {
		t.Fatalf("Status = %q, want %q", state.Status, enum.UnitStatusReviewing)
	}
}

func TestStatusTransitionerDoesNotPromoteLearningWithoutTwoPassingQualities(t *testing.T) {
	transitioner := servicepkg.NewStatusTransitioner()
	schedulerPolicy := policy.DefaultLearningPolicy()
	state := &model.UserUnitState{
		Status:           enum.UnitStatusLearning,
		StrongEventCount: 2,
	}

	if err := transitioner.Recompute(state, []int{4, 2}, schedulerPolicy); err != nil {
		t.Fatalf("Recompute() error = %v", err)
	}

	if state.Status != enum.UnitStatusLearning {
		t.Fatalf("Status = %q, want %q", state.Status, enum.UnitStatusLearning)
	}
}

func TestStatusTransitionerDoesNotMasterWithoutStableRecentPerformance(t *testing.T) {
	transitioner := servicepkg.NewStatusTransitioner()
	schedulerPolicy := policy.DefaultLearningPolicy()
	state := &model.UserUnitState{
		Status:           enum.UnitStatusReviewing,
		IntervalDays:     schedulerPolicy.MasteredIntervalDays,
		ConsecutiveWrong: 0,
	}

	if err := transitioner.Recompute(state, []int{5, 2}, schedulerPolicy); err != nil {
		t.Fatalf("Recompute() error = %v", err)
	}

	if state.Status != enum.UnitStatusReviewing {
		t.Fatalf("Status = %q, want %q", state.Status, enum.UnitStatusReviewing)
	}
}
