// 文件作用：
//   - 定义 UnitKind，表达 semantic.coarse_unit 的种类
//   - 让 Recommendation 在领域层不用直接依赖裸字符串 kind
//
// 输入/输出：
//   - 输入通常来自 mapper 解析数据库中的 kind 字段
//   - 输出用于 CoarseUnitRef、RecommendationItem 和测试断言
//
// 谁调用它：
//   - infrastructure/persistence/mapper/candidate_mapper.go
//   - domain/model/coarse_unit_ref.go
//   - domain/service/recommendation_assembler.go
//
// 它调用谁/传给谁：
//   - 不直接调用其他实现
//   - 作为字段值传给领域模型和最终输出项
package enum

// UnitKind identifies the learning-unit kind stored in semantic.coarse_unit.
type UnitKind string

const (
	UnitKindWord    UnitKind = "word"
	UnitKindPhrase  UnitKind = "phrase"
	UnitKindGrammar UnitKind = "grammar"
)
