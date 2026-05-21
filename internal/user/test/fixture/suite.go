//go:build integration

package fixture

import (
	"context"
	"testing"

	"learning-video-recommendation-system/internal/platform/postgres/pgtest"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Suite struct {
	inner *pgtest.Suite
}

type TestDatabase struct {
	Name string
	Pool *pgxpool.Pool
}

func OpenSuite() (*Suite, error) {
	inner, err := pgtest.OpenSuite(pgtest.Options{
		TempDirPrefix:        "user-integration-*",
		TemplateDatabaseName: "user_template",
		DatabaseNamePrefix:   "user_test",
		SchemaPlan:           schemaPlan(),
	})
	if err != nil {
		return nil, err
	}
	return &Suite{inner: inner}, nil
}

func (s *Suite) Close() error {
	return s.inner.Close()
}

func (s *Suite) CreateTestDatabase(t *testing.T) *TestDatabase {
	t.Helper()
	db := s.inner.CreateTestDatabase(t)
	return &TestDatabase{Name: db.Name, Pool: db.Pool}
}

func (db *TestDatabase) SeedAuthUser(t *testing.T, userID string, email string) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `insert into auth.users (id, email, email_confirmed_at) values ($1, $2, now())`, userID, email); err != nil {
		t.Fatalf("seed auth.users: %v", err)
	}
}
