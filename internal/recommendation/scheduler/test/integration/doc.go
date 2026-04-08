// Package integration 定义 scheduler 的集成测试包说明。
//
// 文件作用：
//   - 说明 integration 目录用于验证真实数据库、真实事务和真实仓储实现
//   - 帮助新人区分这里不是 unit test，也不是跨模块 e2e
//
// 输入/输出：
//   - 输入来自测试夹具构造的真实数据库状态
//   - 输出为对仓储和 usecase 行为的断言结果
//
// 谁调用它：
//   - `go test` 和 `make check`
//
// 它调用谁/传给谁：
//   - 集成测试会调用 fixture、repository、usecase 和 PostgreSQL
package integration
