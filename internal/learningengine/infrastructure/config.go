// 作用：定义并校验 Learning engine 的运行配置，强制要求使用 PostgreSQL 直连。
// 输入/输出：输入是环境变量 DATABASE_URL、SUPABASE_URL；输出是 Config 和 Validate() 的 error。
// 谁调用它：启动装配代码、infrastructure/db.go、integration test、fixture/helpers.go。
// 它调用谁/传给谁：调用标准库 os 读取环境变量；配置对象会传给 db.go 创建连接池。
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

// Config defines runtime configuration used by the learning engine infrastructure.
type Config struct {
	DatabaseURL string
	SupabaseURL string
}

// LoadConfig loads learning engine configuration from environment variables.
func LoadConfig() Config {
	return Config{
		DatabaseURL: os.Getenv(envDatabaseURL),
		SupabaseURL: os.Getenv(envSupabaseURL),
	}
}

// Validate ensures the learning engine is configured to use PostgreSQL direct access.
func (c Config) Validate() error {
	if c.DatabaseURL == "" {
		if c.SupabaseURL != "" {
			return fmt.Errorf("%s must be set for direct PostgreSQL access; %s cannot be used as a fallback", envDatabaseURL, envSupabaseURL)
		}

		return errors.New("DATABASE_URL must be set for learning engine PostgreSQL direct access")
	}

	return nil
}
