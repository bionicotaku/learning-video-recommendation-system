package pgtest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

const adminDBName = "postgres"

// Options configures an embedded Postgres suite.
type Options struct {
	TempDirPrefix        string
	BaseDir              string
	TemplateDatabaseName string
	DatabaseNamePrefix   string
	SchemaPlan           SchemaPlan
}

// Suite owns one embedded Postgres instance and a prepared template database.
type Suite struct {
	postgres       *embeddedpostgres.EmbeddedPostgres
	adminPool      *pgxpool.Pool
	baseDir        string
	cleanupBaseDir bool
	port           uint32
	templateName   string
	dbNamePrefix   string

	mu       sync.Mutex
	nextDBID int64
}

// Database is a cloned test database.
type Database struct {
	suite *Suite
	Name  string
	Pool  *pgxpool.Pool
}

// OpenSuite starts embedded Postgres and prepares the configured template database.
func OpenSuite(options Options) (*Suite, error) {
	var err error
	options, err = normalizeOptions(options)
	if err != nil {
		return nil, err
	}

	baseDir := options.BaseDir
	cleanupBaseDir := false
	if baseDir == "" {
		var err error
		baseDir, err = os.MkdirTemp("", options.TempDirPrefix)
		if err != nil {
			return nil, fmt.Errorf("create temp base dir: %w", err)
		}
		cleanupBaseDir = true
	} else if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create base dir: %w", err)
	}

	port, err := freePort()
	if err != nil {
		if cleanupBaseDir {
			_ = os.RemoveAll(baseDir)
		}
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
		if cleanupBaseDir {
			_ = os.RemoveAll(baseDir)
		}
		return nil, fmt.Errorf("start embedded postgres: %w", err)
	}

	adminPool, err := pgxpool.New(context.Background(), config.GetConnectionURL())
	if err != nil {
		_ = postgres.Stop()
		if cleanupBaseDir {
			_ = os.RemoveAll(baseDir)
		}
		return nil, fmt.Errorf("open admin pool: %w", err)
	}
	if err := waitForDatabase(context.Background(), adminPool); err != nil {
		adminPool.Close()
		_ = postgres.Stop()
		if cleanupBaseDir {
			_ = os.RemoveAll(baseDir)
		}
		return nil, fmt.Errorf("wait for admin pool: %w", err)
	}

	suite := &Suite{
		postgres:       postgres,
		adminPool:      adminPool,
		baseDir:        baseDir,
		cleanupBaseDir: cleanupBaseDir,
		port:           port,
		templateName:   options.TemplateDatabaseName,
		dbNamePrefix:   options.DatabaseNamePrefix,
	}
	if err := suite.prepareTemplateDatabase(options.SchemaPlan); err != nil {
		_ = suite.Close()
		return nil, err
	}
	return suite, nil
}

// Close stops Postgres and removes temporary resources owned by the suite.
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
	if s.cleanupBaseDir && s.baseDir != "" {
		if err := os.RemoveAll(s.baseDir); err != nil {
			errs = append(errs, fmt.Sprintf("remove temp base dir: %v", err))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// CreateTestDatabase clones the template database and registers test cleanup.
func (s *Suite) CreateTestDatabase(t *testing.T) *Database {
	t.Helper()

	db, err := s.OpenDatabase(context.Background(), s.nextDatabaseName())
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(context.Background()); err != nil {
			t.Fatalf("close test database %s: %v", db.Name, err)
		}
	})
	return db
}

// OpenDatabase clones the template database with a caller-provided name.
func (s *Suite) OpenDatabase(ctx context.Context, name string) (*Database, error) {
	if err := validateIdentifier(name, "database name"); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	if _, err := s.adminPool.Exec(ctx, fmt.Sprintf("create database %s template %s", name, s.templateName)); err != nil {
		return nil, fmt.Errorf("create database %s: %w", name, err)
	}

	pool, err := pgxpool.New(context.Background(), s.databaseURL(name))
	if err != nil {
		_ = s.dropDatabase(context.Background(), name)
		return nil, fmt.Errorf("open pool %s: %w", name, err)
	}
	if err := waitForDatabase(context.Background(), pool); err != nil {
		pool.Close()
		_ = s.dropDatabase(context.Background(), name)
		return nil, fmt.Errorf("wait for pool %s: %w", name, err)
	}

	return &Database{
		suite: s,
		Name:  name,
		Pool:  pool,
	}, nil
}

// Close closes and drops the cloned database.
func (db *Database) Close(ctx context.Context) error {
	if db.Pool != nil {
		db.Pool.Close()
	}
	return db.suite.dropDatabase(ctx, db.Name)
}

func (s *Suite) prepareTemplateDatabase(plan SchemaPlan) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.dropDatabase(ctx, s.templateName); err != nil {
		return fmt.Errorf("drop template database: %w", err)
	}
	if _, err := s.adminPool.Exec(ctx, fmt.Sprintf("create database %s", s.templateName)); err != nil {
		return fmt.Errorf("create template database: %w", err)
	}

	templatePool, err := pgxpool.New(context.Background(), s.databaseURL(s.templateName))
	if err != nil {
		return fmt.Errorf("open template pool: %w", err)
	}
	defer templatePool.Close()

	if err := waitForDatabase(context.Background(), templatePool); err != nil {
		return fmt.Errorf("wait for template pool: %w", err)
	}
	if err := plan.Apply(context.Background(), templatePool); err != nil {
		return err
	}
	return nil
}

func (s *Suite) dropDatabase(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	terminateSQL := fmt.Sprintf(
		`select pg_terminate_backend(pid) from pg_stat_activity where datname = '%s' and pid <> pg_backend_pid()`,
		name,
	)
	if _, err := s.adminPool.Exec(ctx, terminateSQL); err != nil {
		return fmt.Errorf("terminate connections for %s: %w", name, err)
	}
	if _, err := s.adminPool.Exec(ctx, fmt.Sprintf("drop database if exists %s", name)); err != nil {
		return fmt.Errorf("drop database %s: %w", name, err)
	}
	return nil
}

func (s *Suite) nextDatabaseName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextDBID++
	return fmt.Sprintf("%s_%d", s.dbNamePrefix, s.nextDBID)
}

func (s *Suite) databaseURL(name string) string {
	return fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/%s?sslmode=disable", s.port, name)
}

func normalizeOptions(options Options) (Options, error) {
	if options.TempDirPrefix == "" {
		options.TempDirPrefix = "pgtest-*"
	}
	if options.TemplateDatabaseName == "" {
		options.TemplateDatabaseName = "pgtest_template"
	}
	if options.DatabaseNamePrefix == "" {
		options.DatabaseNamePrefix = "pgtest_db"
	}
	if err := validateIdentifier(options.TemplateDatabaseName, "template database name"); err != nil {
		return Options{}, err
	}
	if err := validateIdentifier(options.DatabaseNamePrefix, "database name prefix"); err != nil {
		return Options{}, err
	}
	return options, nil
}

func validateIdentifier(value string, field string) error {
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		valid := c == '_' || (c >= 'a' && c <= 'z') || (i > 0 && c >= '0' && c <= '9')
		if !valid {
			return fmt.Errorf("%s must contain only lowercase letters, digits, and underscores, and must start with a lowercase letter or underscore", field)
		}
	}
	if value[0] >= '0' && value[0] <= '9' {
		return fmt.Errorf("%s must contain only lowercase letters, digits, and underscores, and must start with a lowercase letter or underscore", field)
	}
	return nil
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
