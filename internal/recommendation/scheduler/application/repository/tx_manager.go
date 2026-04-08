// 文件作用：
//   - 定义应用层可见的事务边界接口
//   - 让 usecase 能声明“这些写操作必须一起成功或一起失败”，但不依赖具体事务实现
//
// 输入/输出：
//   - 输入：一个携带 ctx 的回调函数
//   - 输出：事务执行成功或失败的 error
//
// 谁调用它：
//   - application/usecase/generate_recommendations.go
//
// 它调用谁/传给谁：
//   - 接口本身不调用其他实现
//   - 由 infrastructure/persistence/tx/pgx_tx_manager.go 实现
package repository

import "context"

// TxManager defines the application-facing transaction boundary.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
