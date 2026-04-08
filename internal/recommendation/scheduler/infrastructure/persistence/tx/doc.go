// Package tx 定义 scheduler 的事务辅助实现。
//
// 文件作用：
//   - 提供 application/repository.TxManager 的基础设施层实现
//   - 负责把 pgx 事务和 context 中的 tx querier 绑定起来
//
// 输入/输出：
//   - 输入来自 usecase 发起的事务回调
//   - 输出为整个事务执行的成功或失败结果
//
// 谁调用它：
//   - 外层组装代码和测试夹具会把这里的实现注入 usecase
//
// 它调用谁/传给谁：
//   - 调用 pgxpool、pgx.Tx、queryctx 和 sqlcgen
package tx
