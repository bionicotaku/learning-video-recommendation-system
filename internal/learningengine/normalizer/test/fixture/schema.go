//go:build integration

package fixture

import "learning-video-recommendation-system/internal/platform/postgres/pgtest"

func schemaPlan() pgtest.SchemaPlan {
	return pgtest.NewSchemaPlan(
		pgtest.SQLFile(pgtest.RepoPath(
			"internal",
			"learningengine",
			"normalizer",
			"infrastructure",
			"persistence",
			"schema",
			"000000_external_refs.sql",
		)),
		pgtest.MigrationDir(pgtest.RepoPath(
			"internal",
			"analytics",
			"infrastructure",
			"migration",
		)),
		pgtest.MigrationDir(pgtest.RepoPath(
			"internal",
			"semantic",
			"infrastructure",
			"migration",
		)),
		pgtest.MigrationDir(pgtest.RepoPath(
			"internal",
			"learningengine",
			"reducer",
			"infrastructure",
			"migration",
		)),
	)
}
