//go:build integration

package fixture

import "learning-video-recommendation-system/internal/platform/postgres/pgtest"

func schemaPlan() pgtest.SchemaPlan {
	return pgtest.NewSchemaPlan(
		pgtest.SQLFile(pgtest.RepoPath(
			"internal",
			"user",
			"infrastructure",
			"persistence",
			"schema",
			"000000_external_refs.sql",
		)),
		pgtest.MigrationDir(pgtest.RepoPath(
			"internal",
			"user",
			"infrastructure",
			"migration",
		)),
	)
}
