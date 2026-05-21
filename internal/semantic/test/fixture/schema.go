//go:build integration

package fixture

import "learning-video-recommendation-system/internal/platform/postgres/pgtest"

func schemaPlan() pgtest.SchemaPlan {
	return pgtest.NewSchemaPlan(
		pgtest.MigrationDir(pgtest.RepoPath(
			"internal",
			"semantic",
			"infrastructure",
			"migration",
		)),
	)
}
