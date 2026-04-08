// 作用：创建和探活 pgx 连接池，为 Learning engine 的 repository 和事务实现提供数据库入口。
// 输入/输出：输入是 context 和 Config；输出是 *pgxpool.Pool 或 ping error。
// 谁调用它：启动装配代码、integration test、fixture/helpers.go。
// 它调用谁/传给谁：调用 pgxpool；返回的 pool 会传给 tx manager、sqlc querier 和 repository 装配逻辑。
package infrastructure

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewDBPool creates a pgx connection pool from DATABASE_URL.
func NewDBPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool from DATABASE_URL: %w", err)
	}

	return pool, nil
}

// PingDB validates the learning engine database connection with a minimal select 1 probe.
func PingDB(ctx context.Context, pool *pgxpool.Pool) error {
	var one int
	if err := pool.QueryRow(ctx, "select 1").Scan(&one); err != nil {
		return fmt.Errorf("ping learning engine database with DATABASE_URL: %w", err)
	}

	if one != 1 {
		return fmt.Errorf("unexpected ping result from DATABASE_URL: got %d", one)
	}

	return nil
}
