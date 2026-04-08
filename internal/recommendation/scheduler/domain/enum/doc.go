// Package enum 定义 scheduler 领域层使用的稳定枚举。
//
// 文件作用：
//   - 收敛 recommend type、unit kind、unit status 这类有限值集合
//   - 避免字符串字面量散落在 scorer、assembler 和 mapper 中
//
// 输入/输出：
//   - 输入来自数据库或上游结构中的字符串值
//   - 输出为领域层统一使用的枚举值
//
// 谁调用它：
//   - mapper、domain/service、domain/model 都会依赖这些枚举
//
// 它调用谁/传给谁：
//   - 不直接调用其他实现
package enum
