//go:build integration

package fixture

import (
	"context"
	"testing"
)

func (db *TestDatabase) SeedUser(t *testing.T, userID string) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `insert into auth.users (id) values ($1) on conflict (id) do nothing`, userID); err != nil {
		t.Fatalf("seed auth.users: %v", err)
	}
}

func (db *TestDatabase) SeedVideo(t *testing.T, videoID string, durationMS int32) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `
		insert into catalog.videos (
			video_id,
			source_clip_key,
			parent_video_name,
			parent_video_slug,
			title,
			language,
			duration_ms,
			video_object_path,
			status,
			visibility_status
		) values (
			$1::uuid,
			'clip-' || $1::text,
			'parent-' || $1::text,
			'parent-' || $1::text,
			'Video ' || $1::text,
			'en',
			$2,
			'portrait_videos/' || $1::text || '.mp4',
			'active',
			'public'
		) on conflict (video_id) do nothing`, videoID, durationMS); err != nil {
		t.Fatalf("seed catalog.videos: %v", err)
	}
}
