// 作用：验证基础设施配置校验逻辑，确保 Learning engine 必须走 DATABASE_URL 直连。
// 输入/输出：输入是不同组合的 Config；输出是 Validate() 返回错误的断言结果。
// 谁调用它：go test、make check。
// 它调用谁/传给谁：调用 infrastructure/config.go；断言只返回给测试框架。
package infrastructure_test

import (
	"testing"

	infra "learning-video-recommendation-system/internal/learningengine/infrastructure"
)

func TestConfigValidateRequiresDatabaseURL(t *testing.T) {
	cfg := infra.Config{}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error")
	}

	if got, want := err.Error(), "DATABASE_URL must be set for learning engine PostgreSQL direct access"; got != want {
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
