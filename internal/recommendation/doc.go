// Package recommendation 是 Recommendation 顶层模块根包。
//
// 文件作用：
//   - 只声明 Recommendation 模块的根级包语义
//   - 告诉阅读者这里是模块边界和子模块容器，不是具体推荐逻辑落点
//
// 输入/输出：
//   - 不直接接收业务输入
//   - 不直接输出推荐结果
//
// 谁调用它：
//   - 主要供阅读代码、生成文档和包级说明时使用
//
// 它调用谁/传给谁：
//   - 不调用任何实现
//   - 当前真实能力下沉在 scheduler 子模块
package recommendation
