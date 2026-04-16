package testutil

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TestDatabase struct {
	postgres *embeddedpostgres.EmbeddedPostgres
	Pool     *pgxpool.Pool
}

func StartPostgres(t *testing.T) *TestDatabase {
	t.Helper()

	port := freePort(t)
	baseDir := t.TempDir()
	config := embeddedpostgres.DefaultConfig().
		Port(uint32(port)).
		Database("learningengine_test").
		Username("postgres").
		Password("postgres").
		RuntimePath(filepath.Join(baseDir, "runtime")).
		DataPath(filepath.Join(baseDir, "data")).
		BinariesPath(filepath.Join(baseDir, "bin"))

	postgres := embeddedpostgres.NewDatabase(config)
	if err := postgres.Start(); err != nil {
		t.Fatalf("start embedded postgres: %v", err)
	}

	dsn := fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/learningengine_test?sslmode=disable", port)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		_ = postgres.Stop()
		t.Fatalf("connect pgx pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		if err := postgres.Stop(); err != nil {
			t.Fatalf("stop embedded postgres: %v", err)
		}
	})

	db := &TestDatabase{
		postgres: postgres,
		Pool:     pool,
	}
	db.ApplyLearningEngineSchema(t)
	return db
}

func (db *TestDatabase) ApplyLearningEngineSchema(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	sqlFiles := []string{
		filepath.Join(repoRoot(t), "internal", "learningengine", "infrastructure", "persistence", "schema", "000000_external_refs.sql"),
	}

	migrationDir := filepath.Join(repoRoot(t), "internal", "learningengine", "infrastructure", "migration")
	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		t.Fatalf("read migration dir: %v", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".up.sql") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		sqlFiles = append(sqlFiles, filepath.Join(migrationDir, name))
	}

	for _, sqlFile := range sqlFiles {
		content, err := os.ReadFile(sqlFile)
		if err != nil {
			t.Fatalf("read sql file %s: %v", sqlFile, err)
		}
		if _, err := db.Pool.Exec(ctx, string(content)); err != nil {
			t.Fatalf("exec sql file %s: %v", sqlFile, err)
		}
	}
}

func (db *TestDatabase) SeedUser(t *testing.T, userID string) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `insert into auth.users (id) values ($1)`, userID); err != nil {
		t.Fatalf("seed auth.users: %v", err)
	}
}

func (db *TestDatabase) SeedCoarseUnit(t *testing.T, unitID int64) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `insert into semantic.coarse_unit (id) values ($1)`, unitID); err != nil {
		t.Fatalf("seed semantic.coarse_unit: %v", err)
	}
}

func (db *TestDatabase) SeedVideo(t *testing.T, videoID string) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `insert into catalog.videos (video_id) values ($1)`, videoID); err != nil {
		t.Fatalf("seed catalog.videos: %v", err)
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate free port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}
