// 文件作用：
//   - 定义 CoarseUnitRef，表达 Recommendation 需要的 coarse unit 轻量只读信息
//   - 避免上层直接依赖 semantic.coarse_unit 的完整数据库结构
//
// 输入/输出：
//   - 输入：mapper 从候选查询结果中解析出的 kind、label、释义等字段
//   - 输出：传给 scorer、assembler 和最终 RecommendationItem
//
// 谁调用它：
//   - infrastructure/persistence/mapper/candidate_mapper.go 负责构造
//   - application/query/candidate.go 持有它
//   - domain/service/recommendation_assembler.go 读取它
//
// 它调用谁/传给谁：
//   - 不直接调用其他实现
//   - 作为领域对象传给 application/query 和 RecommendationItem
package model

import "learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"

// CoarseUnitRef is a lightweight reference to a semantic.coarse_unit record.
type CoarseUnitRef struct {
	CoarseUnitID int64
	Kind         enum.UnitKind
	Label        string
	Pos          string
	EnglishDef   string
	ChineseDef   string
}
