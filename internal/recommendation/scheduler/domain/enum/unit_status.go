// 文件作用：
//   - 定义 UnitStatus，表达 Learning engine 维护的学习状态
//   - 让 scheduler 在优先级提取、组装和测试里复用统一状态枚举
//
// 输入/输出：
//   - 输入通常来自 mapper 对 learning.user_unit_states.status 的解析
//   - 输出用于 LearningStateSnapshot、PriorityZeroExtractor 和 RecommendationItem
//
// 谁调用它：
//   - infrastructure/persistence/mapper/candidate_mapper.go
//   - domain/model/learning_state_snapshot.go
//   - domain/service/priority_zero_extractor.go
//   - domain/service/recommendation_assembler.go
//
// 它调用谁/传给谁：
//   - 不直接调用其他实现
//   - 作为状态字段传给领域对象和结果对象
package enum

// UnitStatus represents the current learning-engine state of a user-unit relation.
type UnitStatus string

const (
	UnitStatusNew       UnitStatus = "new"
	UnitStatusLearning  UnitStatus = "learning"
	UnitStatusReviewing UnitStatus = "reviewing"
	UnitStatusMastered  UnitStatus = "mastered"
	UnitStatusSuspended UnitStatus = "suspended"
)
