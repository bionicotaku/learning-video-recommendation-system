//go:build integration

package fixture

import (
	"context"
	"testing"
)

type UnitCollectionSeed struct {
	CollectionID    string
	Slug            string
	Name            string
	Description     *string
	Category        string
	Status          string
	CoarseUnitCount int32
	WordUnitCount   int32
}

func (db *TestDatabase) SeedUnitCollection(t *testing.T, seed UnitCollectionSeed) {
	t.Helper()
	description := any(nil)
	if seed.Description != nil {
		description = *seed.Description
	}
	if _, err := db.Pool.Exec(context.Background(), `
		insert into semantic.unit_collections (
			collection_id,
			slug,
			name,
			description,
			category,
			status,
			coarse_unit_count,
			word_unit_count
		) values (
			$1::uuid,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8
		)`, seed.CollectionID, seed.Slug, seed.Name, description, seed.Category, seed.Status, seed.CoarseUnitCount, seed.WordUnitCount); err != nil {
		t.Fatalf("seed semantic.unit_collections: %v", err)
	}
}
