//go:build integration

package fixture

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func OpenPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	port := freePort(t)
	baseDir := tempBaseDir(t)
	config := embeddedpostgres.DefaultConfig().
		Version(embeddedpostgres.V17).
		Port(port).
		Database("postgres").
		Username("postgres").
		Password("postgres").
		DataPath(filepath.Join(baseDir, "data")).
		RuntimePath(filepath.Join(baseDir, "run")).
		CachePath(filepath.Join(baseDir, "cache"))

	db := embeddedpostgres.NewDatabase(config)
	if err := db.Start(); err != nil {
		t.Fatalf("start embedded postgres: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Stop()
	})

	pool, err := pgxpool.New(context.Background(), config.GetConnectionURL())
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	if err := WaitForDatabase(context.Background(), pool); err != nil {
		t.Fatalf("wait for database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func BeginTestTx(t *testing.T, pool *pgxpool.Pool) pgx.Tx {
	t.Helper()

	tx, err := pool.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	t.Cleanup(func() {
		_ = tx.Rollback(context.Background())
	})
	return tx
}

func EnsureRecommendationStep1Schema(ctx context.Context, db execer) error {
	statements := []string{
		`create schema if not exists catalog`,
		`create schema if not exists recommendation`,
		`create table if not exists catalog.video_user_states (
			user_id uuid not null,
			video_id uuid not null,
			last_watched_at timestamptz,
			watch_count integer not null default 0,
			completed_count integer not null default 0,
			last_watch_ratio numeric(6,5),
			max_watch_ratio numeric(6,5)
		)`,
		`create table if not exists recommendation.user_unit_serving_states (
			user_id uuid not null,
			coarse_unit_id bigint not null,
			last_served_at timestamptz,
			last_run_id uuid,
			served_count integer not null default 0,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			primary key (user_id, coarse_unit_id)
		)`,
		`create table if not exists recommendation.user_video_serving_states (
			user_id uuid not null,
			video_id uuid not null,
			last_served_at timestamptz,
			last_run_id uuid,
			served_count integer not null default 0,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			primary key (user_id, video_id)
		)`,
		`create table if not exists recommendation.video_recommendation_runs (
			run_id uuid primary key,
			user_id uuid not null,
			request_context jsonb not null default '{}'::jsonb,
			session_mode text,
			selector_mode text,
			planner_snapshot jsonb not null default '{}'::jsonb,
			lane_budget_snapshot jsonb not null default '{}'::jsonb,
			candidate_summary jsonb not null default '{}'::jsonb,
			underfilled boolean not null default false,
			result_count integer not null default 0,
			created_at timestamptz not null default now()
		)`,
		`create table if not exists recommendation.video_recommendation_items (
			run_id uuid not null,
			rank integer not null,
			video_id uuid not null,
			score numeric(10,4) not null default 0,
			primary_lane text,
			dominant_bucket text,
			dominant_unit_id bigint,
			reason_codes text[] not null default '{}',
			covered_hard_review_count integer not null default 0,
			covered_new_now_count integer not null default 0,
			covered_soft_review_count integer not null default 0,
			covered_near_future_count integer not null default 0,
			best_evidence_sentence_index integer,
			best_evidence_span_index integer,
			best_evidence_start_ms integer,
			best_evidence_end_ms integer,
			created_at timestamptz not null default now(),
			primary key (run_id, rank)
		)`,
	}

	for _, statement := range statements {
		if _, err := db.Exec(ctx, statement); err != nil {
			return fmt.Errorf("exec schema statement: %w", err)
		}
	}

	return nil
}

func tempBaseDir(t *testing.T) string {
	t.Helper()

	baseDir, err := os.MkdirTemp("", "recommendation-integration-*")
	if err != nil {
		t.Fatalf("mkdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(baseDir)
	})
	return baseDir
}

func freePort(t *testing.T) uint32 {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve tcp port: %v", err)
	}
	defer listener.Close()

	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatal("unexpected listener address type")
	}

	return uint32(address.Port)
}

func WaitForDatabase(ctx context.Context, pool *pgxpool.Pool) error {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := pool.Ping(ctx); err == nil {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("database did not become ready before deadline")
}
