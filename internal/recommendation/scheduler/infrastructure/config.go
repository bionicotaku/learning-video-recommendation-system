// 文件作用：
//   - 定义 scheduler 基础设施层的运行配置
//   - 强制 Recommendation 使用 DATABASE_URL 直连 PostgreSQL
//
// 输入/输出：
//   - 输入：环境变量 DATABASE_URL 和 SUPABASE_URL
//   - 输出：Config 结构，以及 Validate 的校验结果
//
// 谁调用它：
//   - infrastructure/db.go
//   - 集成测试 fixture.NewTestPool
//   - config 单元测试
//
// 它调用谁/传给谁：
//   - 调用 os.Getenv 读取环境变量
//   - 把校验后的配置传给 NewDBPool
package infrastructure

import (
	"errors"
	"fmt"
	"os"
)

const (
	envDatabaseURL = "DATABASE_URL"
	envSupabaseURL = "SUPABASE_URL"
)

// Config defines runtime configuration used by the recommendation infrastructure.
type Config struct {
	DatabaseURL string
	SupabaseURL string
}

// LoadConfig loads recommendation configuration from environment variables.
func LoadConfig() Config {
	return Config{
		DatabaseURL: os.Getenv(envDatabaseURL),
		SupabaseURL: os.Getenv(envSupabaseURL),
	}
}

// Validate ensures Recommendation is configured to use PostgreSQL direct access.
func (c Config) Validate() error {
	if c.DatabaseURL == "" {
		if c.SupabaseURL != "" {
			return fmt.Errorf("%s must be set for direct PostgreSQL access; %s cannot be used as a fallback", envDatabaseURL, envSupabaseURL)
		}

		return errors.New("DATABASE_URL must be set for recommendation PostgreSQL direct access")
	}

	return nil
}
