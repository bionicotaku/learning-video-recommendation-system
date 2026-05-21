//go:build integration

package fixture

import (
	"context"
	"testing"
)

func (db *TestDatabase) SeedUser(t *testing.T, userID string) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `insert into auth.users (id) values ($1)`, userID); err != nil {
		t.Fatalf("seed auth.users: %v", err)
	}
}

func (db *TestDatabase) SeedCoarseUnit(t *testing.T, unitID int64) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `
		insert into semantic.coarse_unit (
			id,
			kind,
			label,
			lang,
			chinese_label,
			english_label,
			status,
			version,
			fine_unit_ids,
			original_defs
		) values (
			$1::bigint,
			'word',
			'unit-' || $1::text,
			'en',
			'unit-cn-' || $1::text,
			'unit ' || $1::text,
			'active',
			1,
			'{}'::bigint[],
			'{}'::text[]
		)`, unitID); err != nil {
		t.Fatalf("seed semantic.coarse_unit: %v", err)
	}
}

func (db *TestDatabase) SeedVideo(t *testing.T, videoID string) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `insert into catalog.videos (video_id) values ($1)`, videoID); err != nil {
		t.Fatalf("seed catalog.videos: %v", err)
	}
}

func (db *TestDatabase) SeedUnitCollection(t *testing.T, collectionID string, slug string, name string, status string) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `
		insert into semantic.unit_collections (
			collection_id,
			slug,
			name,
			status,
			coarse_unit_count
		) values (
			$1::uuid,
			$2,
			$3,
			$4,
			0
		)`, collectionID, slug, name, status); err != nil {
		t.Fatalf("seed semantic.unit_collections: %v", err)
	}
}

func (db *TestDatabase) SeedUnitCollectionMember(t *testing.T, collectionID string, coarseUnitID int64, sortOrder int) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `
		insert into semantic.unit_collection_members (
			collection_id,
			coarse_unit_id,
			sort_order,
			target_priority
		) values (
			$1::uuid,
			$2,
			$3,
			0
		)`, collectionID, coarseUnitID, sortOrder); err != nil {
		t.Fatalf("seed semantic.unit_collection_members: %v", err)
	}
}

func (db *TestDatabase) SetCollectionCounts(t *testing.T, collectionID string, coarseUnitCount int32, _ int32) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `
		update semantic.unit_collections
		set coarse_unit_count = $2
		where collection_id = $1::uuid`, collectionID, coarseUnitCount); err != nil {
		t.Fatalf("update semantic.unit_collections count: %v", err)
	}
}

func (db *TestDatabase) SeedUserUnitState(t *testing.T, userID string, coarseUnitID int64, targetSource string, targetSourceRefID string, isTarget bool, status string, progressPercent float64) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `
		insert into learning.user_unit_states (
			user_id,
			coarse_unit_id,
			is_target,
			target_source,
			target_source_ref_id,
			target_priority,
			status,
			progress_percent
		) values (
			$1::uuid,
			$2,
			$3,
			$4,
			$5,
			0,
			$6,
			$7
		)
		on conflict (user_id, coarse_unit_id) do update
		set
			is_target = excluded.is_target,
			target_source = excluded.target_source,
			target_source_ref_id = excluded.target_source_ref_id,
			status = excluded.status,
			progress_percent = excluded.progress_percent`, userID, coarseUnitID, isTarget, targetSource, targetSourceRefID, status, progressPercent); err != nil {
		t.Fatalf("seed learning.user_unit_states: %v", err)
	}
}
