//go:build integration

package fixture

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	adminDBName    = "postgres"
	templateDBName = "recommendation_template"
)

type execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type Suite struct {
	postgres  *embeddedpostgres.EmbeddedPostgres
	adminPool *pgxpool.Pool
	baseDir   string
	port      uint32

	mu       sync.Mutex
	nextDBID int64
}

type TestDatabase struct {
	suite *Suite
	Name  string
	Pool  *pgxpool.Pool
}

func OpenSuite() (*Suite, error) {
	baseDir, err := os.MkdirTemp("", "recommendation-integration-*")
	if err != nil {
		return nil, fmt.Errorf("create temp base dir: %w", err)
	}

	port, err := freePort()
	if err != nil {
		_ = os.RemoveAll(baseDir)
		return nil, fmt.Errorf("allocate free port: %w", err)
	}

	config := embeddedpostgres.DefaultConfig().
		Version(embeddedpostgres.V17).
		Port(port).
		Database(adminDBName).
		Username("postgres").
		Password("postgres").
		RuntimePath(filepath.Join(baseDir, "runtime")).
		DataPath(filepath.Join(baseDir, "data")).
		CachePath(filepath.Join(baseDir, "cache"))

	postgres := embeddedpostgres.NewDatabase(config)
	if err := postgres.Start(); err != nil {
		_ = os.RemoveAll(baseDir)
		return nil, fmt.Errorf("start embedded postgres: %w", err)
	}

	adminPool, err := pgxpool.New(context.Background(), config.GetConnectionURL())
	if err != nil {
		_ = postgres.Stop()
		_ = os.RemoveAll(baseDir)
		return nil, fmt.Errorf("open admin pool: %w", err)
	}
	if err := waitForDatabase(context.Background(), adminPool); err != nil {
		adminPool.Close()
		_ = postgres.Stop()
		_ = os.RemoveAll(baseDir)
		return nil, fmt.Errorf("wait for admin pool: %w", err)
	}

	suite := &Suite{
		postgres:  postgres,
		adminPool: adminPool,
		baseDir:   baseDir,
		port:      port,
	}
	if err := suite.prepareTemplateDatabase(); err != nil {
		_ = suite.Close()
		return nil, err
	}
	return suite, nil
}

func (s *Suite) Close() error {
	var errs []string
	if s.adminPool != nil {
		s.adminPool.Close()
	}
	if s.postgres != nil {
		if err := s.postgres.Stop(); err != nil {
			errs = append(errs, fmt.Sprintf("stop embedded postgres: %v", err))
		}
	}
	if s.baseDir != "" {
		if err := os.RemoveAll(s.baseDir); err != nil {
			errs = append(errs, fmt.Sprintf("remove temp base dir: %v", err))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func (s *Suite) CreateTestDatabase(t *testing.T) *TestDatabase {
	t.Helper()

	name := s.nextDatabaseName()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if _, err := s.adminPool.Exec(ctx, fmt.Sprintf("create database %s template %s", name, templateDBName)); err != nil {
		t.Fatalf("create test database %s: %v", name, err)
	}

	pool, err := pgxpool.New(context.Background(), s.databaseURL(name))
	if err != nil {
		t.Fatalf("open test pool %s: %v", name, err)
	}
	if err := waitForDatabase(context.Background(), pool); err != nil {
		pool.Close()
		t.Fatalf("wait for test pool %s: %v", name, err)
	}

	db := &TestDatabase{
		suite: s,
		Name:  name,
		Pool:  pool,
	}

	t.Cleanup(func() {
		db.close(t)
	})
	return db
}

func (db *TestDatabase) SeedUser(t *testing.T, userID string) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `insert into auth.users (id) values ($1) on conflict (id) do nothing`, userID); err != nil {
		t.Fatalf("seed auth.users: %v", err)
	}
}

func (db *TestDatabase) SeedCoarseUnit(t *testing.T, unitID int64) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `insert into semantic.coarse_unit (id) values ($1) on conflict (id) do nothing`, unitID); err != nil {
		t.Fatalf("seed semantic.coarse_unit: %v", err)
	}
}

func (db *TestDatabase) SeedVideo(t *testing.T, videoID string) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `insert into catalog.videos (video_id, duration_ms, status, visibility_status) values ($1, 120000, 'active', 'public') on conflict (video_id) do nothing`, videoID); err != nil {
		t.Fatalf("seed catalog.videos: %v", err)
	}
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

func (db *TestDatabase) close(t *testing.T) {
	t.Helper()

	if db.Pool != nil {
		db.Pool.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	terminateSQL := fmt.Sprintf(
		`select pg_terminate_backend(pid) from pg_stat_activity where datname = '%s' and pid <> pg_backend_pid()`,
		db.Name,
	)
	if _, err := db.suite.adminPool.Exec(ctx, terminateSQL); err != nil {
		t.Fatalf("terminate connections for %s: %v", db.Name, err)
	}
	if _, err := db.suite.adminPool.Exec(ctx, fmt.Sprintf("drop database if exists %s", db.Name)); err != nil {
		t.Fatalf("drop test database %s: %v", db.Name, err)
	}
}

func (s *Suite) prepareTemplateDatabase() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := s.adminPool.Exec(ctx, fmt.Sprintf("drop database if exists %s", templateDBName)); err != nil {
		return fmt.Errorf("drop template database: %w", err)
	}
	if _, err := s.adminPool.Exec(ctx, fmt.Sprintf("create database %s", templateDBName)); err != nil {
		return fmt.Errorf("create template database: %w", err)
	}

	templatePool, err := pgxpool.New(context.Background(), s.databaseURL(templateDBName))
	if err != nil {
		return fmt.Errorf("open template pool: %w", err)
	}
	defer templatePool.Close()

	if err := waitForDatabase(context.Background(), templatePool); err != nil {
		return fmt.Errorf("wait for template pool: %w", err)
	}
	if err := applyRecommendationSchema(context.Background(), templatePool); err != nil {
		return err
	}
	return nil
}

func (s *Suite) nextDatabaseName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextDBID++
	return fmt.Sprintf("recommendation_test_%d", s.nextDBID)
}

func (s *Suite) databaseURL(name string) string {
	return fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/%s?sslmode=disable", s.port, name)
}

func applyRecommendationSchema(ctx context.Context, db execer) error {
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

func waitForDatabase(ctx context.Context, pool *pgxpool.Pool) error {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := pool.Ping(ctx); err == nil {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("database did not become ready before deadline")
}

func freePort() (uint32, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("reserve tcp port: %w", err)
	}
	defer listener.Close()

	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected listener address type")
	}

	return uint32(address.Port), nil
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
