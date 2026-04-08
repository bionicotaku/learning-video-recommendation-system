// 文件作用：
//   - 验证 scheduler 基础设施层能否用 DATABASE_URL 正常建连接池并完成探活
//   - 为所有依赖真实数据库的集成测试兜底
//
// 输入/输出：
//   - 输入：环境变量中的 DATABASE_URL
//   - 输出：对 NewDBPool 和 PingDB 成功执行的断言
//
// 谁调用它：
//   - `go test` 和 `make check`
//
// 它调用谁/传给谁：
//   - 调用 infrastructure.LoadConfig / NewDBPool / PingDB
//   - 间接调用 PostgreSQL
package infrastructure_test

import (
	"context"
	"testing"
	"time"

	infra "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure"
)

func TestNewDBPoolAndPing(t *testing.T) {
	cfg := infra.LoadConfig()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	t.Log("using DATABASE_URL for recommendation PostgreSQL direct-access test")

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
