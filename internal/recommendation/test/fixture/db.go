//go:build integration

package fixture

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
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
	if err := execSQLFile(ctx, db, repoPath(
		"internal",
		"recommendation",
		"infrastructure",
		"persistence",
		"schema",
		"000000_external_refs.sql",
	)); err != nil {
		return err
	}

	if _, err := db.Exec(ctx, `drop materialized view if exists recommendation.v_unit_video_inventory`); err != nil {
		return fmt.Errorf("drop inventory stub view: %w", err)
	}
	if _, err := db.Exec(ctx, `drop materialized view if exists recommendation.v_recommendable_video_units`); err != nil {
		return fmt.Errorf("drop recommendable stub view: %w", err)
	}

	migrations, err := migrationFiles(repoPath("internal", "recommendation", "infrastructure", "migration"))
	if err != nil {
		return err
	}
	for _, path := range migrations {
		if err := execSQLFile(ctx, db, path); err != nil {
			return err
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

func execSQLFile(ctx context.Context, db execer, path string) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read sql file %s: %w", path, err)
	}
	if _, err := db.Exec(ctx, string(contents)); err != nil {
		return fmt.Errorf("exec sql file %s: %w", path, err)
	}
	return nil
}

func migrationFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migration dir: %w", err)
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}

	sort.Slice(files, func(i, j int) bool {
		return migrationVersion(files[i]) < migrationVersion(files[j])
	})
	return files, nil
}

func migrationVersion(path string) int {
	base := filepath.Base(path)
	version, err := strconv.Atoi(strings.SplitN(base, "_", 2)[0])
	if err != nil {
		return 0
	}
	return version
}

func repoPath(parts ...string) string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve fixture path")
	}
	base := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../../.."))
	all := append([]string{base}, parts...)
	return filepath.Join(all...)
}
