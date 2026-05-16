package pgtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// Execer is the minimal database execution contract used by schema plans.
type Execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// SchemaPlan is an ordered set of schema setup steps.
type SchemaPlan struct {
	Steps []SchemaStep
}

// SchemaStep is one schema setup step.
type SchemaStep struct {
	kind        schemaStepKind
	description string
	path        string
	sql         string
}

type schemaStepKind string

const (
	schemaStepSQLFile      schemaStepKind = "sql_file"
	schemaStepSQLText      schemaStepKind = "sql_text"
	schemaStepMigrationDir schemaStepKind = "migration_dir"
)

// NewSchemaPlan creates an ordered schema plan.
func NewSchemaPlan(steps ...SchemaStep) SchemaPlan {
	return SchemaPlan{Steps: steps}
}

// SQLFile adds a SQL file execution step.
func SQLFile(path string) SchemaStep {
	return SchemaStep{kind: schemaStepSQLFile, path: path}
}

// SQLText adds an inline SQL execution step.
func SQLText(description string, sql string) SchemaStep {
	return SchemaStep{kind: schemaStepSQLText, description: description, sql: sql}
}

// MigrationDir adds all .up.sql files in a migration directory in version order.
func MigrationDir(path string) SchemaStep {
	return SchemaStep{kind: schemaStepMigrationDir, path: path}
}

// Apply executes the schema plan against db.
func (p SchemaPlan) Apply(ctx context.Context, db Execer) error {
	for _, step := range p.Steps {
		if err := step.apply(ctx, db); err != nil {
			return err
		}
	}
	return nil
}

func (s SchemaStep) apply(ctx context.Context, db Execer) error {
	switch s.kind {
	case schemaStepSQLFile:
		return ExecSQLFile(ctx, db, s.path)
	case schemaStepSQLText:
		if err := ExecSQLText(ctx, db, s.sql); err != nil {
			return fmt.Errorf("exec inline sql %s: %w", s.description, err)
		}
		return nil
	case schemaStepMigrationDir:
		files, err := MigrationFiles(s.path)
		if err != nil {
			return err
		}
		for _, path := range files {
			if err := ExecSQLFile(ctx, db, path); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown schema step kind %q", s.kind)
	}
}

// ExecSQLFile reads and executes one SQL file.
func ExecSQLFile(ctx context.Context, db Execer, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read sql file %s: %w", path, err)
	}
	if err := ExecSQLText(ctx, db, string(content)); err != nil {
		return fmt.Errorf("exec sql file %s: %w", path, err)
	}
	return nil
}

// ExecSQLText executes one SQL string.
func ExecSQLText(ctx context.Context, db Execer, sql string) error {
	if _, err := db.Exec(ctx, sql); err != nil {
		return err
	}
	return nil
}

// MigrationFiles returns .up.sql files in a migration directory in numeric version order.
func MigrationFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migration dir %s: %w", dir, err)
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}

	sort.Slice(files, func(i, j int) bool {
		left := migrationVersion(files[i])
		right := migrationVersion(files[j])
		if left == right {
			return files[i] < files[j]
		}
		return left < right
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

// RepoRoot returns the repository root path.
func RepoRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve pgtest caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", ".."))
}

// RepoPath joins path parts relative to the repository root.
func RepoPath(parts ...string) string {
	all := append([]string{RepoRoot()}, parts...)
	return filepath.Join(all...)
}
