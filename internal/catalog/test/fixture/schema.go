//go:build integration

package fixture

import "learning-video-recommendation-system/internal/platform/postgres/pgtest"

func schemaPlan() pgtest.SchemaPlan {
	return pgtest.NewSchemaPlan(
		pgtest.SQLFile(pgtest.RepoPath(
			"internal",
			"catalog",
			"infrastructure",
			"persistence",
			"schema",
			"000000_external_refs.sql",
		)),
		pgtest.MigrationDir(pgtest.RepoPath(
			"internal",
			"catalog",
			"infrastructure",
			"migration",
		)),
		pgtest.SQLFile(pgtest.RepoPath(
			"internal",
			"analytics",
			"infrastructure",
			"migration",
			"000001_create_analytics_schema.up.sql",
		)),
		pgtest.SQLFile(pgtest.RepoPath(
			"internal",
			"analytics",
			"infrastructure",
			"migration",
			"000003_create_video_watch_events.up.sql",
		)),
	)
}
