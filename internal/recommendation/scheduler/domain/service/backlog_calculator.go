// 文件作用：
//   - 定义 BacklogCalculator，把 due review 候选数转换成 review backlog
//   - 当前规则很简单，但保留独立服务方便未来扩展 backlog 定义
//
// 输入/输出：
//   - 输入：reviewBacklog 候选数
//   - 输出：归一化后的 backlog 值，负数会被钳为 0
//
// 谁调用它：
//   - application/usecase/generate_recommendations.go
//   - unit test 会直接验证其边界行为
//
// 它调用谁/传给谁：
//   - 不调用外部实现
//   - 计算结果传给 QuotaAllocator
package service

type BacklogCalculator interface {
	Compute(reviewBacklog int) int
}

type backlogCalculator struct{}

func NewBacklogCalculator() BacklogCalculator {
	return backlogCalculator{}
}

func (backlogCalculator) Compute(reviewBacklog int) int {
	if reviewBacklog < 0 {
		return 0
	}

	return reviewBacklog
}
