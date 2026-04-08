// 作用：声明 application 层可见的事务边界接口，让 use case 不依赖 pgx 细节。
// 输入/输出：输入是 context 和一个事务回调；输出是回调执行后的 error。
// 谁调用它：record_learning_events.go、replay_user_states.go 两个 use case。
// 它调用谁/传给谁：接口本身不调用其他文件；由 infrastructure/persistence/tx/pgx_tx_manager.go 实现。
package repository

import "context"

// TxManager defines the application-facing transaction boundary.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
