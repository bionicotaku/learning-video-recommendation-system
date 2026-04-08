// Package domain 是 scheduler 的领域层。
//
// 文件作用：
//   - 承载纯推荐规则、枚举和值对象
//   - 明确 domain 不处理 SQL、事务和数据库连接
//
// 输入/输出：
//   - 输入来自 application/query 和 domain/model
//   - 输出为评分结果、quota 结果和最终 RecommendationBatch
//
// 谁调用它：
//   - application/usecase 会统一调用 domain/service
//
// 它调用谁/传给谁：
//   - domain/service 会消费 application/query 和 domain/model
package domain
