//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/catalog/domain/model"
	catalogrepo "learning-video-recommendation-system/internal/catalog/infrastructure/persistence/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestFeedLookupReaderListFeedVideosByIDs(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()

	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	visibleID := "11111111-1111-1111-1111-111111111111"
	noStatsID := "22222222-2222-2222-2222-222222222222"
	inactiveID := "33333333-3333-3333-3333-333333333333"
	privateID := "44444444-4444-4444-4444-444444444444"
	futureID := "55555555-5555-5555-5555-555555555555"

	seedFeedVideo(t, db.Pool, visibleID, "Visible title", "Visible description", "hls/visible/master.m3u8", "covers/visible.webp", "active", "public", nil)
	seedFeedVideo(t, db.Pool, noStatsID, "No stats title", "", "https://cdn.example.com/hls/no-stats/master.m3u8", "", "active", "public", nil)
	seedFeedVideo(t, db.Pool, inactiveID, "Inactive", "", "hls/inactive/master.m3u8", "", "inactive", "public", nil)
	seedFeedVideo(t, db.Pool, privateID, "Private", "", "hls/private/master.m3u8", "", "active", "private", nil)
	future := time.Now().UTC().Add(24 * time.Hour)
	seedFeedVideo(t, db.Pool, futureID, "Future", "", "hls/future/master.m3u8", "", "active", "public", &future)

	if _, err := db.Pool.Exec(ctx, `
		insert into auth.users (id, email)
		values ($1, 'feed-user@example.com')`, userID); err != nil {
		t.Fatalf("seed auth user: %v", err)
	}
	if _, err := db.Pool.Exec(ctx, `
		insert into catalog.video_engagement_stats (video_id, view_count, like_count, favorite_count)
		values ($1, 12, 3, 2)`, visibleID); err != nil {
		t.Fatalf("seed stats: %v", err)
	}
	if _, err := db.Pool.Exec(ctx, `
		insert into catalog.video_transcripts (
			video_id,
			transcript_object_path,
			transcript_checksum,
			sentence_count,
			semantic_span_count,
			mapped_span_count,
			unmapped_span_count,
			mapped_span_ratio
		)
		values ($1, 'transcripts/visible.json', 'checksum-visible', 1, 1, 1, 0, 1.0)`, visibleID); err != nil {
		t.Fatalf("seed transcript: %v", err)
	}
	if _, err := db.Pool.Exec(ctx, `
		insert into catalog.video_user_states (user_id, video_id, has_liked, has_bookmarked)
		values ($1, $2, true, true)`, userID, visibleID); err != nil {
		t.Fatalf("seed user state: %v", err)
	}

	reader := catalogrepo.NewFeedLookupReader(db.Pool)
	videos, err := reader.ListFeedVideosByIDs(ctx, userID, []string{visibleID, noStatsID, inactiveID, privateID, futureID})
	if err != nil {
		t.Fatalf("list feed videos: %v", err)
	}

	if len(videos) != 2 {
		t.Fatalf("expected 2 visible videos, got %d: %+v", len(videos), videos)
	}

	byID := make(map[string]model.FeedVideoDisplay, len(videos))
	for _, video := range videos {
		byID[video.VideoID] = video
	}

	visible, ok := byID[visibleID]
	if !ok {
		t.Fatalf("visible video missing from result: %+v", videos)
	}
	if visible.Title != "Visible title" || visible.Description != "Visible description" || visible.VideoObjectPath != "hls/visible/master.m3u8" {
		t.Fatalf("unexpected visible video metadata: %+v", visible)
	}
	if visible.CoverImageURL == nil || *visible.CoverImageURL != "covers/visible.webp" {
		t.Fatalf("unexpected cover image: %+v", visible.CoverImageURL)
	}
	if visible.TranscriptObjectPath == nil || *visible.TranscriptObjectPath != "transcripts/visible.json" {
		t.Fatalf("unexpected transcript path: %+v", visible.TranscriptObjectPath)
	}
	if visible.ViewCount != 12 || visible.LikeCount != 3 || visible.FavoriteCount != 2 {
		t.Fatalf("unexpected stats: %+v", visible)
	}
	if !visible.HasLiked || !visible.HasFavorited {
		t.Fatalf("unexpected interaction state: %+v", visible)
	}

	noStats, ok := byID[noStatsID]
	if !ok {
		t.Fatalf("no-stats video missing from result: %+v", videos)
	}
	if noStats.Description != "" || noStats.CoverImageURL != nil {
		t.Fatalf("empty description/cover should be normalized: %+v", noStats)
	}
	if noStats.TranscriptObjectPath != nil {
		t.Fatalf("missing transcript should return nil: %+v", noStats.TranscriptObjectPath)
	}
	if noStats.ViewCount != 0 || noStats.LikeCount != 0 || noStats.FavoriteCount != 0 {
		t.Fatalf("missing stats should default to zero: %+v", noStats)
	}
	if noStats.HasLiked || noStats.HasFavorited {
		t.Fatalf("missing user state should default to false: %+v", noStats)
	}
}

func TestFeedLookupReaderListUnitLabelsByIDs(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()

	if _, err := db.Pool.Exec(ctx, `
		insert into semantic.coarse_unit (id, label, status)
		values
			(101, 'serendipity', 'active'),
			(102, 'deprecated', 'inactive')`); err != nil {
		t.Fatalf("seed coarse units: %v", err)
	}

	reader := catalogrepo.NewFeedLookupReader(db.Pool)
	labels, err := reader.ListUnitLabelsByIDs(ctx, []int64{101, 102, 999})
	if err != nil {
		t.Fatalf("list unit labels: %v", err)
	}

	if len(labels) != 1 {
		t.Fatalf("expected 1 active label, got %d: %+v", len(labels), labels)
	}
	if labels[0].CoarseUnitID != 101 || labels[0].Text != "serendipity" {
		t.Fatalf("unexpected label: %+v", labels[0])
	}
}

func seedFeedVideo(
	t *testing.T,
	pool *pgxpool.Pool,
	videoID string,
	title string,
	description string,
	hlsPath string,
	coverImageURL string,
	status string,
	visibility string,
	publishAt *time.Time,
) {
	t.Helper()

	var descriptionValue any
	if description != "" {
		descriptionValue = description
	}
	var coverValue any
	if coverImageURL != "" {
		coverValue = coverImageURL
	}
	if _, err := pool.Exec(context.Background(), `
		insert into catalog.videos (
			video_id,
			source_clip_key,
			parent_video_name,
			parent_video_slug,
			title,
			description,
			language,
			duration_ms,
			video_object_path,
			thumbnail_url,
			status,
			visibility_status,
			publish_at
		) values (
			$1::uuid,
			'feed-' || $1::text,
			'parent-' || $1::text,
			'parent-' || $1::text,
			$2,
			$3,
			'en',
			90000,
			$4,
			$5,
			$6,
			$7,
			$8
		)`, videoID, title, descriptionValue, hlsPath, coverValue, status, visibility, publishAt); err != nil {
		t.Fatalf("seed feed video: %v", err)
	}
}
