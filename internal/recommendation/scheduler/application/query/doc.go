// Package query 定义 scheduler 应用层查询模型。
//
// 文件作用：
//   - 承接 repository 从数据库读出的候选结构
//   - 为后续打分、排序和组装提供稳定输入模型
//
// 输入/输出：
//   - 输入是 infrastructure/repository 从 SQL 查询和 mapper 转出的结果
//   - 输出是传给 domain/service 的候选对象
//
// 谁调用它：
//   - application/usecase
//   - domain/service 中的 scorer、assembler、priority extractor
//
// 它调用谁/传给谁：
//   - 不直接调用其他实现
//   - 作为 usecase 与 domain/service 之间的中间数据结构
package query
