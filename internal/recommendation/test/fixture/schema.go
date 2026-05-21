//go:build integration

package fixture

import "learning-video-recommendation-system/internal/platform/postgres/pgtest"

func schemaPlan() pgtest.SchemaPlan {
	return pgtest.NewSchemaPlan(
		pgtest.SQLFile(pgtest.RepoPath(
			"internal",
			"recommendation",
			"infrastructure",
			"persistence",
			"schema",
			"000000_external_refs.sql",
		)),
		pgtest.SQLText("drop placeholder recommendation materialized views", `
			drop materialized view if exists recommendation.v_unit_video_inventory;
			drop materialized view if exists recommendation.v_video_unit_recall_index;
			drop materialized view if exists recommendation.v_recommendable_video_units;
		`),
		pgtest.MigrationDir(pgtest.RepoPath(
			"internal",
			"recommendation",
			"infrastructure",
			"migration",
		)),
	)
}
