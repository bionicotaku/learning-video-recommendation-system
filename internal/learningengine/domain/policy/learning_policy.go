// 作用：定义学习策略对象及其默认值，统一 mastered 阈值、初始间隔和最小 EF 下限。
// 输入/输出：输入无；输出是 LearningPolicy 结构和 DefaultLearningPolicy() 默认策略。
// 谁调用它：record/replay usecase、user_state_rebuilder、user_unit_reducer、各类测试。
// 它调用谁/传给谁：不主动调用其他文件；返回的策略会传给 reducer、SM2Updater、StatusTransitioner 等规则组件。
package policy

type LearningPolicy struct {
	MasteredIntervalDays float64
	InitialIntervals     []float64
	MinEaseFactor        float64
}

const (
	DefaultMasteredIntervalDays = 21
	DefaultMinEaseFactor        = 1.3
)

var defaultInitialIntervals = []float64{1, 3, 6}

// DefaultLearningPolicy returns the MVP learning policy defaults from the design doc.
func DefaultLearningPolicy() LearningPolicy {
	intervals := make([]float64, len(defaultInitialIntervals))
	copy(intervals, defaultInitialIntervals)

	return LearningPolicy{
		MasteredIntervalDays: DefaultMasteredIntervalDays,
		InitialIntervals:     intervals,
		MinEaseFactor:        DefaultMinEaseFactor,
	}
}
