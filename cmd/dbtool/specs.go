package main

import (
	"fmt"
	"path/filepath"
)

type moduleSpec struct {
	Name          string
	MigrationDir  string
	TrackingTable string
}

func moduleSpecs() map[string]moduleSpec {
	return map[string]moduleSpec{
		"analytics": {
			Name:          "analytics",
			MigrationDir:  filepath.FromSlash("internal/analytics/infrastructure/migration"),
			TrackingTable: "analytics_schema_migrations",
		},
		"semantic": {
			Name:          "semantic",
			MigrationDir:  filepath.FromSlash("internal/semantic/infrastructure/migration"),
			TrackingTable: "semantic_schema_migrations",
		},
		"catalog": {
			Name:          "catalog",
			MigrationDir:  filepath.FromSlash("internal/catalog/infrastructure/migration"),
			TrackingTable: "catalog_schema_migrations",
		},
		"learningengine": {
			Name:          "learningengine",
			MigrationDir:  filepath.FromSlash("internal/learningengine/reducer/infrastructure/migration"),
			TrackingTable: "learningengine_schema_migrations",
		},
		"recommendation": {
			Name:          "recommendation",
			MigrationDir:  filepath.FromSlash("internal/recommendation/infrastructure/migration"),
			TrackingTable: "recommendation_schema_migrations",
		},
		"user": {
			Name:          "user",
			MigrationDir:  filepath.FromSlash("internal/user/infrastructure/migration"),
			TrackingTable: "user_schema_migrations",
		},
	}
}

func resolveModule(name string) (moduleSpec, error) {
	spec, ok := moduleSpecs()[name]
	if !ok {
		return moduleSpec{}, fmt.Errorf("unknown module %q", name)
	}
	return spec, nil
}

func refreshTargets() []string {
	return []string{
		"recommendation.v_video_unit_recall_index",
		"recommendation.v_unit_video_inventory",
	}
}
