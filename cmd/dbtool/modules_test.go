package main

import "testing"

func TestModuleSpecsContainExpectedRegistry(t *testing.T) {
	specs := moduleSpecs()

	if len(specs) != 3 {
		t.Fatalf("expected 3 module specs, got %d", len(specs))
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
}

func TestRefreshTargetsOnlyRecommendationMaterializedViews(t *testing.T) {
	targets := refreshTargets()

	if len(targets) != 2 {
		t.Fatalf("expected 2 refresh targets, got %d", len(targets))
	}

	expected := map[string]struct{}{
		"recommendation.v_recommendable_video_units": {},
		"recommendation.v_unit_video_inventory":      {},
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
