// 文件作用：
//   - 定义 RecommendType，表达推荐项是 review 还是 new
//   - 为 assembler、持久化映射和测试提供统一枚举
//
// 输入/输出：
//   - 输入通常来自 domain/service 组装结果
//   - 输出为 RecommendationItem 和落库参数中的 recommend_type
//
// 谁调用它：
//   - domain/service/recommendation_assembler.go
//   - infrastructure/persistence/mapper/scheduler_run_mapper.go
//   - 各类测试
//
// 它调用谁/传给谁：
//   - 不直接调用其他实现
//   - 作为字段值传给 model.RecommendationItem 和 SQL 参数
package enum

// RecommendType identifies whether a recommendation item is review or new.
type RecommendType string

const (
	RecommendTypeReview RecommendType = "review"
	RecommendTypeNew    RecommendType = "new"
)
