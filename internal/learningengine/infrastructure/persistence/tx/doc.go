// 作用：提供 Learning engine 的事务实现，使 application 层只感知 TxManager 接口。
// 输入/输出：输入是数据库连接池和事务回调；输出是事务执行结果。
// 谁调用它：application/usecase 的装配代码、fixture/helpers.go。
// 它调用谁/传给谁：调用 pgx transaction 和 queryctx；事务上下文最终传给 repository。
package tx
