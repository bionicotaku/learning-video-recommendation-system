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
