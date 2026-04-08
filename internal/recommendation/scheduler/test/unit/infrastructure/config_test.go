// 文件作用：
//   - 验证 scheduler 基础设施配置校验逻辑
//   - 防止 DATABASE_URL 必填约束被意外放松
//
// 输入/输出：
//   - 输入：测试中构造的 Config
//   - 输出：对 Validate 错误信息的断言
//
// 谁调用它：
//   - `go test` 和 `make check`
//
// 它调用谁/传给谁：
//   - 直接调用 infrastructure/config.go 的 Validate
package infrastructure_test

import (
	"testing"

	infra "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure"
)

func TestConfigValidateRequiresDatabaseURL(t *testing.T) {
	cfg := infra.Config{}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error")
	}

	if got, want := err.Error(), "DATABASE_URL must be set for recommendation PostgreSQL direct access"; got != want {
		t.Fatalf("Validate() error = %q, want %q", got, want)
	}
}

func TestConfigValidateDoesNotFallbackToSupabaseURL(t *testing.T) {
	cfg := infra.Config{SupabaseURL: "https://example.supabase.co"}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error")
	}

	if got, want := err.Error(), "DATABASE_URL must be set for direct PostgreSQL access; SUPABASE_URL cannot be used as a fallback"; got != want {
		t.Fatalf("Validate() error = %q, want %q", got, want)
	}
}
