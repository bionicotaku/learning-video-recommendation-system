// 作用：声明 Learning engine 集成测试包，承载真实数据库、真实事务和真实仓储编排验证。
// 输入/输出：输入来自 go test 和测试环境变量；输出是集成测试结果。
// 谁调用它：go test、make check。
// 它调用谁/传给谁：调用 fixture、infrastructure、usecase、repository 等真实实现。
package integration
