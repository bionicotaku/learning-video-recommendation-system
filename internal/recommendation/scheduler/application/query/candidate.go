// 文件作用：
//   - 定义 review/new 候选以及打分后的候选结构
//   - 统一描述“学习状态快照 + coarse unit 信息 + serving state”这一组输入
//
// 输入/输出：
//   - 输入：来自 infrastructure/repository 读库后经 mapper 转换的数据
//   - 输出：提供给 domain/service 做打分、优先级提取和最终组装
//
// 谁调用它：
//   - infrastructure/persistence/repository/learning_state_snapshot_read_repo.go 负责构造
//   - application/usecase/generate_recommendations.go 负责收集和传递
//   - domain/service/*.go 负责消费
//
// 它调用谁/传给谁：
//   - 不直接调用其他实现
//   - 作为 usecase 与 scorer/assembler 之间的中间模型
package query

import "learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"

// ReviewCandidate is a due review candidate returned by the recommendation query layer.
type ReviewCandidate struct {
	State   model.LearningStateSnapshot
	Unit    model.CoarseUnitRef
	Serving model.UserUnitServingState
}

// NewCandidate is a new-learning candidate returned by the recommendation query layer.
type NewCandidate struct {
	State   model.LearningStateSnapshot
	Unit    model.CoarseUnitRef
	Serving model.UserUnitServingState
}

// ScoredReviewCandidate is a review candidate with its computed score and reasons.
type ScoredReviewCandidate struct {
	Candidate   ReviewCandidate
	Score       float64
	ReasonCodes []string
}

// ScoredNewCandidate is a new candidate with its computed score and reasons.
type ScoredNewCandidate struct {
	Candidate   NewCandidate
	Score       float64
	ReasonCodes []string
}
