// 文件作用：
//   - 提供 scheduler 基础设施层的数据库连接池创建和探活能力
//   - 把 pgxpool 的创建逻辑集中在一处，避免在各处散落连接初始化细节
//
// 输入/输出：
//   - 输入：context 和经过校验的 Config
//   - 输出：*pgxpool.Pool 或错误；PingDB 输出连接探活结果
//
// 谁调用它：
//   - 集成测试 fixture.NewTestPool
//   - 未来的生产级组装入口也应从这里创建连接池
//
// 它调用谁/传给谁：
//   - 调用 pgxpool.ParseConfig / NewWithConfig
//   - 调用 PostgreSQL 执行 select 1 探活
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

// PingDB validates the recommendation database connection with a minimal select 1 probe.
func PingDB(ctx context.Context, pool *pgxpool.Pool) error {
	var one int
	if err := pool.QueryRow(ctx, "select 1").Scan(&one); err != nil {
		return fmt.Errorf("ping recommendation database with DATABASE_URL: %w", err)
	}

	if one != 1 {
		return fmt.Errorf("unexpected ping result from DATABASE_URL: got %d", one)
	}

	return nil
}
