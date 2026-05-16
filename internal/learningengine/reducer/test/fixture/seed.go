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
