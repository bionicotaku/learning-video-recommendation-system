// Package dto 定义 scheduler 应用层输出 DTO。
//
// 文件作用：
//   - 把 usecase 的输出包成稳定返回结构
//   - 避免调用方直接依赖 usecase 内部临时变量或持久化细节
//
// 输入/输出：
//   - 输入来自 application/usecase 的最终结果
//   - 输出为交给上层调用方的 DTO
//
// 谁调用它：
//   - application/usecase 会构造这里的结构体
//
// 它调用谁/传给谁：
//   - 不直接调用其他实现
//   - 主要返回给调用 usecase 的上层代码和测试
package dto
