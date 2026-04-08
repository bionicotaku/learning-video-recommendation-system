// Package application 是 Recommendation scheduler 的应用层。
//
// 文件作用：
//   - 说明 application 目录只负责用例编排，不负责领域规则和持久化细节
//   - 帮助新人快速确认 command/dto/query/repository/usecase 的职责边界
//
// 输入/输出：
//   - 输入来自上层调用方构造的 command
//   - 输出为 usecase 返回的 dto
//
// 谁调用它：
//   - 上层调用方通常通过 usecase 进入这一层
//
// 它调用谁/传给谁：
//   - application/usecase 会调用 domain/service 和 infrastructure/repository
package application
