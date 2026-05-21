package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModuleSpecsContainExpectedRegistry(t *testing.T) {
	specs := moduleSpecs()

	if len(specs) != 6 {
		t.Fatalf("expected 6 module specs, got %d", len(specs))
	}

	analytics, ok := specs["analytics"]
	if !ok {
		t.Fatalf("expected analytics module spec to exist")
	}
	if analytics.TrackingTable != "analytics_schema_migrations" {
		t.Fatalf("unexpected analytics tracking table: %s", analytics.TrackingTable)
	}

	semantic, ok := specs["semantic"]
	if !ok {
		t.Fatalf("expected semantic module spec to exist")
	}
	if semantic.TrackingTable != "semantic_schema_migrations" {
		t.Fatalf("unexpected semantic tracking table: %s", semantic.TrackingTable)
	}

	catalog, ok := specs["catalog"]
	if !ok {
		t.Fatalf("expected catalog module spec to exist")
	}
	if catalog.TrackingTable != "catalog_schema_migrations" {
		t.Fatalf("unexpected catalog tracking table: %s", catalog.TrackingTable)
	}

	learning, ok := specs["learningengine"]
	if !ok {
		t.Fatalf("expected learningengine module spec to exist")
	}
	if learning.TrackingTable != "learningengine_schema_migrations" {
		t.Fatalf("unexpected learningengine tracking table: %s", learning.TrackingTable)
	}

	recommendation, ok := specs["recommendation"]
	if !ok {
		t.Fatalf("expected recommendation module spec to exist")
	}
	if recommendation.TrackingTable != "recommendation_schema_migrations" {
		t.Fatalf("unexpected recommendation tracking table: %s", recommendation.TrackingTable)
	}

	user, ok := specs["user"]
	if !ok {
		t.Fatalf("expected user module spec to exist")
	}
	if user.TrackingTable != "user_schema_migrations" {
		t.Fatalf("unexpected user tracking table: %s", user.TrackingTable)
	}
}

func TestRefreshTargetsOnlyRecommendationMaterializedViews(t *testing.T) {
	targets := refreshTargets()

	if len(targets) != 2 {
		t.Fatalf("expected 2 refresh targets, got %d", len(targets))
	}

	expected := map[string]struct{}{
		"recommendation.v_video_unit_recall_index": {},
		"recommendation.v_unit_video_inventory":    {},
	}

	for _, target := range targets {
		if _, ok := expected[target]; !ok {
			t.Fatalf("unexpected refresh target %q", target)
		}
		delete(expected, target)
	}

	if len(expected) != 0 {
		t.Fatalf("missing refresh targets: %#v", expected)
	}
}

func TestResolveModuleRejectsUnknownNames(t *testing.T) {
	if _, err := resolveModule("unknown"); err == nil {
		t.Fatalf("expected unknown module to be rejected")
	}
}

func TestRecommendationMigrationSixOnlyDropsLegacyRecallView(t *testing.T) {
	up := readRepoFile(t, "internal", "recommendation", "infrastructure", "migration", "000006_replace_recommendable_video_units_with_recall_index.up.sql")
	down := readRepoFile(t, "internal", "recommendation", "infrastructure", "migration", "000006_replace_recommendable_video_units_with_recall_index.down.sql")

	for _, unexpected := range []string{
		"create materialized view if not exists recommendation.v_video_unit_recall_index",
		"create materialized view if not exists recommendation.v_unit_video_inventory",
		"drop materialized view if exists recommendation.v_video_unit_recall_index",
		"drop materialized view if exists recommendation.v_unit_video_inventory",
	} {
		if strings.Contains(up, unexpected) {
			t.Fatalf("migration 000006 up must not own current recall views, found %q", unexpected)
		}
		if strings.Contains(down, unexpected) {
			t.Fatalf("migration 000006 down must not drop current recall views, found %q", unexpected)
		}
	}
	if !strings.Contains(up, "drop materialized view if exists recommendation.v_recommendable_video_units") {
		t.Fatalf("migration 000006 up should only remove the legacy recommendable view")
	}
}

func readRepoFile(t *testing.T, pathParts ...string) string {
	t.Helper()
	parts := append([]string{"..", ".."}, pathParts...)
	content, err := os.ReadFile(filepath.Join(parts...))
	if err != nil {
		t.Fatalf("read repo file: %v", err)
	}
	return string(content)
}
