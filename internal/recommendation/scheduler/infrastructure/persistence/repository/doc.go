// Package repository 定义 scheduler 持久化层的仓储实现。
//
// 文件作用：
//   - 实现 application/repository 中声明的接口
//   - 把 SQL 执行、tx querier 选择和 mapper 拼装封装在仓储层
//
// 输入/输出：
//   - 输入来自 application/usecase 发出的读写请求
//   - 输出为候选数据、落库结果或错误
//
// 谁调用它：
//   - 外层组装代码会把这些实现注入 usecase
//   - fixture.NewGenerateUseCase 就是当前最完整组装入口
//
// 它调用谁/传给谁：
//   - 调用 sqlcgen、mapper、queryctx 和 PostgreSQL
package repository
