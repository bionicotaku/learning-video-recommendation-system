// Package scheduler 是 Recommendation 当前唯一已落地的子模块。
//
// 文件作用：
//   - 声明 scheduler 子模块的包语义
//   - 告诉阅读者真正的推荐实现都在这个子模块之下
//
// 输入/输出：
//   - 不直接接收业务输入
//   - 不直接输出推荐结果
//
// 谁调用它：
//   - 主要供阅读代码和生成包级文档时使用
//
// 它调用谁/传给谁：
//   - 不调用具体实现
//   - 具体能力由 application/domain/infrastructure/test 分层承载
package scheduler
