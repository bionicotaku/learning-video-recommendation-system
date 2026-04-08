// Package service 定义 scheduler 的领域服务。
//
// 文件作用：
//   - 承载 backlog、quota、打分、优先级提取和批次组装等纯业务规则
//   - 保证这些规则不依赖 SQL、pgx 和事务实现
//
// 输入/输出：
//   - 输入来自 application/query 和 domain/model
//   - 输出为打分结果、quota 结果和 RecommendationBatch
//
// 谁调用它：
//   - application/usecase/generate_recommendations.go
//
// 它调用谁/传给谁：
//   - 领域服务之间少量共享 helper
//   - 最终结果返回给 usecase
package service
