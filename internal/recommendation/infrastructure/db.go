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
