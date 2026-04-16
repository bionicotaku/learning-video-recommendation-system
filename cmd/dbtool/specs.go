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
		"catalog": {
			Name:          "catalog",
			MigrationDir:  filepath.FromSlash("internal/catalog/infrastructure/migration"),
			TrackingTable: "catalog_schema_migrations",
		},
		"learningengine": {
			Name:          "learningengine",
			MigrationDir:  filepath.FromSlash("internal/learningengine/infrastructure/migration"),
			TrackingTable: "learningengine_schema_migrations",
		},
		"recommendation": {
			Name:          "recommendation",
			MigrationDir:  filepath.FromSlash("internal/recommendation/infrastructure/migration"),
			TrackingTable: "recommendation_schema_migrations",
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
		"recommendation.v_recommendable_video_units",
		"recommendation.v_unit_video_inventory",
	}
}
