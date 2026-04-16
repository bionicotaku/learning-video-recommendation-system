package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type migrationFile struct {
	Version  int64
	UpPath   string
	DownPath string
}

type migrationStatus struct {
	CurrentVersion int64
	Applied        []int64
	Pending        []int64
}

func discoverMigrations(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	versions := make(map[int64]*migrationFile)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		version, direction, ok := parseMigrationFilename(name)
		if !ok {
			continue
		}

		record, ok := versions[version]
		if !ok {
			record = &migrationFile{Version: version}
			versions[version] = record
		}

		path := filepath.Join(dir, name)
		switch direction {
		case "up":
			record.UpPath = path
		case "down":
			record.DownPath = path
		}
	}

	result := make([]migrationFile, 0, len(versions))
	for _, migration := range versions {
		if migration.UpPath == "" || migration.DownPath == "" {
			return nil, fmt.Errorf("migration version %d is missing an up or down file", migration.Version)
		}
		result = append(result, *migration)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})

	return result, nil
}

func parseMigrationFilename(name string) (int64, string, bool) {
	parts := strings.Split(name, "_")
	if len(parts) < 2 {
		return 0, "", false
	}

	version, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", false
	}

	switch {
	case strings.HasSuffix(name, ".up.sql"):
		return version, "up", true
	case strings.HasSuffix(name, ".down.sql"):
		return version, "down", true
	default:
		return 0, "", false
	}
}

func ensureTrackingTable(ctx context.Context, conn *pgxpool.Pool, tableName string) error {
	query := fmt.Sprintf(
		`create table if not exists %s (
			version bigint primary key,
			applied_at timestamptz not null default now()
		)`,
		pgx.Identifier{tableName}.Sanitize(),
	)
	_, err := conn.Exec(ctx, query)
	return err
}

func appliedVersions(ctx context.Context, conn *pgxpool.Pool, tableName string) ([]int64, error) {
	query := fmt.Sprintf(
		"select version from %s order by version asc",
		pgx.Identifier{tableName}.Sanitize(),
	)
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []int64
	for rows.Next() {
		var version int64
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}

	return versions, rows.Err()
}

func migrationPlan(ctx context.Context, conn *pgxpool.Pool, spec moduleSpec) (migrationStatus, []migrationFile, error) {
	if err := ensureTrackingTable(ctx, conn, spec.TrackingTable); err != nil {
		return migrationStatus{}, nil, err
	}

	migrations, err := discoverMigrations(spec.MigrationDir)
	if err != nil {
		return migrationStatus{}, nil, err
	}

	applied, err := appliedVersions(ctx, conn, spec.TrackingTable)
	if err != nil {
		return migrationStatus{}, nil, err
	}

	appliedSet := make(map[int64]struct{}, len(applied))
	for _, version := range applied {
		appliedSet[version] = struct{}{}
	}

	status := migrationStatus{
		Applied: applied,
	}
	if len(applied) > 0 {
		status.CurrentVersion = applied[len(applied)-1]
	}

	for _, migration := range migrations {
		if _, ok := appliedSet[migration.Version]; !ok {
			status.Pending = append(status.Pending, migration.Version)
		}
	}

	return status, migrations, nil
}

func migrateUp(ctx context.Context, conn *pgxpool.Pool, spec moduleSpec) error {
	status, migrations, err := migrationPlan(ctx, conn, spec)
	if err != nil {
		return err
	}

	appliedSet := make(map[int64]struct{}, len(status.Applied))
	for _, version := range status.Applied {
		appliedSet[version] = struct{}{}
	}

	for _, migration := range migrations {
		if _, ok := appliedSet[migration.Version]; ok {
			continue
		}
		if err := executeMigrationFile(ctx, conn, spec.TrackingTable, migration.Version, migration.UpPath, true); err != nil {
			return err
		}
		fmt.Printf("applied %s migration %d\n", spec.Name, migration.Version)
	}

	return nil
}

func migrateDown(ctx context.Context, conn *pgxpool.Pool, spec moduleSpec, steps int) error {
	if steps <= 0 {
		return fmt.Errorf("steps must be greater than 0")
	}

	status, migrations, err := migrationPlan(ctx, conn, spec)
	if err != nil {
		return err
	}
	if len(status.Applied) == 0 {
		return nil
	}

	downByVersion := make(map[int64]string, len(migrations))
	for _, migration := range migrations {
		downByVersion[migration.Version] = migration.DownPath
	}

	toRollback := status.Applied
	if len(toRollback) > steps {
		toRollback = toRollback[len(toRollback)-steps:]
	}

	for i := len(toRollback) - 1; i >= 0; i-- {
		version := toRollback[i]
		downPath, ok := downByVersion[version]
		if !ok {
			return fmt.Errorf("missing down migration for version %d", version)
		}
		if err := executeMigrationFile(ctx, conn, spec.TrackingTable, version, downPath, false); err != nil {
			return err
		}
		fmt.Printf("rolled back %s migration %d\n", spec.Name, version)
	}

	return nil
}

func executeMigrationFile(ctx context.Context, conn *pgxpool.Pool, trackingTable string, version int64, path string, apply bool) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, string(contents)); err != nil {
		return err
	}

	if apply {
		query := fmt.Sprintf(
			"insert into %s (version) values ($1) on conflict (version) do nothing",
			pgx.Identifier{trackingTable}.Sanitize(),
		)
		if _, err := tx.Exec(ctx, query, version); err != nil {
			return err
		}
	} else {
		query := fmt.Sprintf(
			"delete from %s where version = $1",
			pgx.Identifier{trackingTable}.Sanitize(),
		)
		if _, err := tx.Exec(ctx, query, version); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func printVersion(ctx context.Context, conn *pgxpool.Pool, spec moduleSpec) error {
	status, _, err := migrationPlan(ctx, conn, spec)
	if err != nil {
		return err
	}
	fmt.Println(status.CurrentVersion)
	return nil
}

func printStatus(ctx context.Context, conn *pgxpool.Pool, spec moduleSpec) error {
	status, _, err := migrationPlan(ctx, conn, spec)
	if err != nil {
		return err
	}

	fmt.Printf("module=%s current=%d applied=%d pending=%d\n", spec.Name, status.CurrentVersion, len(status.Applied), len(status.Pending))
	return nil
}

func refreshRecommendation(ctx context.Context, conn *pgxpool.Pool) error {
	for _, target := range refreshTargets() {
		query := fmt.Sprintf("refresh materialized view %s", target)
		if _, err := conn.Exec(ctx, query); err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}
			return fmt.Errorf("refresh %s: %w", target, err)
		}
		fmt.Printf("refreshed %s\n", target)
	}
	return nil
}
