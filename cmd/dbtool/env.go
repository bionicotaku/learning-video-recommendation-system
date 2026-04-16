package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func loadDatabaseURL() (string, error) {
	if value := os.Getenv("DATABASE_URL"); value != "" {
		return value, nil
	}

	_ = godotenv.Load()

	if value := os.Getenv("DATABASE_URL"); value != "" {
		return value, nil
	}

	return "", fmt.Errorf("DATABASE_URL is not set")
}

func openPool(ctx context.Context) (*pgxpool.Pool, error) {
	databaseURL, err := loadDatabaseURL()
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	return pool, nil
}
