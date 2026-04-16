package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: dbtool <migrate|refresh> ...")
	}

	switch args[0] {
	case "migrate":
		return runMigrate(ctx, args[1:])
	case "refresh":
		return runRefresh(ctx, args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runMigrate(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: dbtool migrate <up|down|version|status> --module=<name>")
	}

	switch args[0] {
	case "up":
		fs := flag.NewFlagSet("up", flag.ContinueOnError)
		moduleName := fs.String("module", "", "module name")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return withModule(ctx, *moduleName, func(conn dbConn, spec moduleSpec) error {
			return migrateUp(ctx, conn.pool, spec)
		})
	case "down":
		fs := flag.NewFlagSet("down", flag.ContinueOnError)
		moduleName := fs.String("module", "", "module name")
		steps := fs.Int("steps", 1, "number of migrations to roll back")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return withModule(ctx, *moduleName, func(conn dbConn, spec moduleSpec) error {
			return migrateDown(ctx, conn.pool, spec, *steps)
		})
	case "version":
		fs := flag.NewFlagSet("version", flag.ContinueOnError)
		moduleName := fs.String("module", "", "module name")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return withModule(ctx, *moduleName, func(conn dbConn, spec moduleSpec) error {
			return printVersion(ctx, conn.pool, spec)
		})
	case "status":
		fs := flag.NewFlagSet("status", flag.ContinueOnError)
		moduleName := fs.String("module", "", "module name")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return withModule(ctx, *moduleName, func(conn dbConn, spec moduleSpec) error {
			return printStatus(ctx, conn.pool, spec)
		})
	default:
		return fmt.Errorf("unknown migrate command %q", args[0])
	}
}

func runRefresh(ctx context.Context, args []string) error {
	if len(args) != 1 || args[0] != "recommendation" {
		return fmt.Errorf("usage: dbtool refresh recommendation")
	}

	conn, err := openConn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	return refreshRecommendation(ctx, conn.pool)
}

type dbConn struct {
	pool *pgxpool.Pool
}

func (c dbConn) Close() {
	if c.pool != nil {
		c.pool.Close()
	}
}

func openConn(ctx context.Context) (dbConn, error) {
	pool, err := openPool(ctx)
	if err != nil {
		return dbConn{}, err
	}
	return dbConn{pool: pool}, nil
}

func withModule(ctx context.Context, moduleName string, fn func(conn dbConn, spec moduleSpec) error) error {
	if moduleName == "" {
		return fmt.Errorf("module is required")
	}

	spec, err := resolveModule(moduleName)
	if err != nil {
		return err
	}

	conn, err := openConn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	return fn(conn, spec)
}
