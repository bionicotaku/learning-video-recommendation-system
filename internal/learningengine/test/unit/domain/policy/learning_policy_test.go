// 作用：验证默认 LearningPolicy 是否符合文档约束，并确保默认 interval slice 不会共享底层数据。
// 输入/输出：输入是 DefaultLearningPolicy() 返回值；输出是测试断言结果。
// 谁调用它：go test、make check。
// 它调用谁/传给谁：调用 domain/policy/learning_policy.go；断言只返回给测试框架。
package policy_test

import (
	"testing"

	policypkg "learning-video-recommendation-system/internal/learningengine/domain/policy"
)

func TestDefaultLearningPolicyUsesDocumentedDefaults(t *testing.T) {
	got := policypkg.DefaultLearningPolicy()

	if got.MasteredIntervalDays != policypkg.DefaultMasteredIntervalDays {
		t.Fatalf("MasteredIntervalDays = %v, want %v", got.MasteredIntervalDays, policypkg.DefaultMasteredIntervalDays)
	}
	if got.MinEaseFactor != policypkg.DefaultMinEaseFactor {
		t.Fatalf("MinEaseFactor = %v, want %v", got.MinEaseFactor, policypkg.DefaultMinEaseFactor)
	}

	wantIntervals := []float64{1, 3, 6}
	if len(got.InitialIntervals) != len(wantIntervals) {
		t.Fatalf("len(InitialIntervals) = %d, want %d", len(got.InitialIntervals), len(wantIntervals))
	}
	for index, want := range wantIntervals {
		if got.InitialIntervals[index] != want {
			t.Fatalf("InitialIntervals[%d] = %v, want %v", index, got.InitialIntervals[index], want)
		}
	}
}

func TestDefaultLearningPolicyReturnsCopiedIntervals(t *testing.T) {
	first := policypkg.DefaultLearningPolicy()
	first.InitialIntervals[0] = 99

	second := policypkg.DefaultLearningPolicy()
	if second.InitialIntervals[0] != 1 {
		t.Fatalf("InitialIntervals[0] = %v, want 1 after prior mutation", second.InitialIntervals[0])
	}
}
