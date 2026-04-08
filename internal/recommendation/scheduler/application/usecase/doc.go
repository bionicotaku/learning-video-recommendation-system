// Package usecase 定义 scheduler 的应用层用例。
//
// 文件作用：
//   - 放置 Recommendation 对外暴露的主业务入口
//   - 负责串起读取、规则计算、结果组装和事务写入
//
// 输入/输出：
//   - 输入来自 command
//   - 输出为 dto
//
// 谁调用它：
//   - 上层业务组装代码
//   - 集成测试和场景测试
//
// 它调用谁/传给谁：
//   - 调用 application/repository 接口和 domain/service
package usecase
