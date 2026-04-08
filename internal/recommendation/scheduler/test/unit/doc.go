// Package unit 定义 scheduler 单元测试包说明。
//
// 文件作用：
//   - 说明 unit 目录只验证纯函数、纯规则和局部基础设施校验
//   - 帮助新人区分 unit 与 integration 的边界
//
// 输入/输出：
//   - 输入来自测试内直接构造的对象
//   - 输出为对单个函数或小范围行为的断言
//
// 谁调用它：
//   - `go test` 和 `make check`
//
// 它调用谁/传给谁：
//   - 单元测试会直接调用 domain/service 或 infrastructure/config
package unit
