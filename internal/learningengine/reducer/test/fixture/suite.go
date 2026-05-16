//go:build integration

package fixture

import (
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
		TempDirPrefix:        "learningengine-integration-*",
		TemplateDatabaseName: "learningengine_template",
		DatabaseNamePrefix:   "learningengine_test",
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
	return &TestDatabase{
		Name: db.Name,
		Pool: db.Pool,
	}
}
