// 作用：验证 Learning engine 的数据库配置和 pgx 连接池在真实 DATABASE_URL 下可以建连并 ping 成功。
// 输入/输出：输入是环境变量中的 DATABASE_URL；输出是测试断言结果。
// 谁调用它：go test、make check。
// 它调用谁/传给谁：调用 infrastructure/config.go 和 infrastructure/db.go；不向外传递业务结果。
package infrastructure_test

import (
	"context"
	"testing"
	"time"

	infra "learning-video-recommendation-system/internal/learningengine/infrastructure"
)

func TestNewDBPoolAndPing(t *testing.T) {
	cfg := infra.LoadConfig()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	t.Log("using DATABASE_URL for learning engine PostgreSQL direct-access test")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := infra.NewDBPool(ctx, cfg)
	if err != nil {
		t.Fatalf("NewDBPool() error = %v", err)
	}
	defer pool.Close()

	if err := infra.PingDB(ctx, pool); err != nil {
		t.Fatalf("PingDB() error = %v", err)
	}
}
