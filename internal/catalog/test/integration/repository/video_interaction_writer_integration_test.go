//go:build integration

package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	apprepo "learning-video-recommendation-system/internal/catalog/application/repository"
	"learning-video-recommendation-system/internal/catalog/domain/model"
	catalogrepo "learning-video-recommendation-system/internal/catalog/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/catalog/test/fixture"
)

func TestVideoInteractionWriterLikeIdempotency(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	db.SeedUser(t, userID)
	db.SeedVideo(t, videoID, 100000)
	writer := catalogrepo.NewVideoInteractionWriter(db.Pool)

	first, err := writer.SetVideoLike(context.Background(), likeCommand(true))
	if err != nil {
		t.Fatalf("first like: %v", err)
	}
	if !first.HasLiked || first.LikeCount != 1 {
		t.Fatalf("unexpected first like result: %+v", first)
	}

	second, err := writer.SetVideoLike(context.Background(), likeCommand(true))
	if err != nil {
		t.Fatalf("second like: %v", err)
	}
	if !second.HasLiked || second.LikeCount != 1 {
		t.Fatalf("duplicate like should not increment: %+v", second)
	}

	unset, err := writer.SetVideoLike(context.Background(), likeCommand(false))
	if err != nil {
		t.Fatalf("unset like: %v", err)
	}
	if unset.HasLiked || unset.LikeCount != 0 {
		t.Fatalf("unexpected unlike result: %+v", unset)
	}

	repeatedUnset, err := writer.SetVideoLike(context.Background(), likeCommand(false))
	if err != nil {
		t.Fatalf("repeat unset like: %v", err)
	}
	if repeatedUnset.HasLiked || repeatedUnset.LikeCount != 0 {
		t.Fatalf("duplicate unlike should not decrement: %+v", repeatedUnset)
	}

	state := readInteractionState(t, db, userID, videoID)
	if state.HasLiked || state.LikedAt != nil {
		t.Fatalf("unexpected persisted like state: %+v", state)
	}
	stats := readInteractionStats(t, db, videoID)
	if stats.LikeCount != 0 {
		t.Fatalf("unexpected like count: %+v", stats)
	}
}

func TestVideoInteractionWriterFavoriteIdempotency(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	db.SeedUser(t, userID)
	db.SeedVideo(t, videoID, 100000)
	writer := catalogrepo.NewVideoInteractionWriter(db.Pool)

	first, err := writer.SetVideoFavorite(context.Background(), favoriteCommand(true))
	if err != nil {
		t.Fatalf("first favorite: %v", err)
	}
	if !first.HasFavorited || first.FavoriteCount != 1 {
		t.Fatalf("unexpected first favorite result: %+v", first)
	}

	second, err := writer.SetVideoFavorite(context.Background(), favoriteCommand(true))
	if err != nil {
		t.Fatalf("second favorite: %v", err)
	}
	if !second.HasFavorited || second.FavoriteCount != 1 {
		t.Fatalf("duplicate favorite should not increment: %+v", second)
	}

	unset, err := writer.SetVideoFavorite(context.Background(), favoriteCommand(false))
	if err != nil {
		t.Fatalf("unset favorite: %v", err)
	}
	if unset.HasFavorited || unset.FavoriteCount != 0 {
		t.Fatalf("unexpected unfavorite result: %+v", unset)
	}

	repeatedUnset, err := writer.SetVideoFavorite(context.Background(), favoriteCommand(false))
	if err != nil {
		t.Fatalf("repeat unset favorite: %v", err)
	}
	if repeatedUnset.HasFavorited || repeatedUnset.FavoriteCount != 0 {
		t.Fatalf("duplicate unfavorite should not decrement: %+v", repeatedUnset)
	}

	state := readInteractionState(t, db, userID, videoID)
	if state.HasBookmarked || state.BookmarkedAt != nil {
		t.Fatalf("unexpected persisted favorite state: %+v", state)
	}
	stats := readInteractionStats(t, db, videoID)
	if stats.FavoriteCount != 0 {
		t.Fatalf("unexpected favorite count: %+v", stats)
	}
}

func TestVideoInteractionWriterIgnoresStaleLikeStateChanges(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	db.SeedUser(t, userID)
	db.SeedVideo(t, videoID, 100000)
	writer := catalogrepo.NewVideoInteractionWriter(db.Pool)

	newer := likeCommandAt(true, time.Date(2026, 5, 23, 12, 10, 0, 0, time.UTC))
	if _, err := writer.SetVideoLike(context.Background(), newer); err != nil {
		t.Fatalf("newer like: %v", err)
	}

	stale := likeCommandAt(false, time.Date(2026, 5, 23, 12, 5, 0, 0, time.UTC))
	result, err := writer.SetVideoLike(context.Background(), stale)
	if err != nil {
		t.Fatalf("stale unlike: %v", err)
	}
	if !result.HasLiked || result.LikeCount != 1 {
		t.Fatalf("stale unlike should return current liked state/count: %+v", result)
	}

	state := readInteractionState(t, db, userID, videoID)
	if !state.HasLiked || state.LikedAt == nil || !state.LikedAt.Equal(newer.OccurredAt) || state.LikeStateUpdatedAt == nil || !state.LikeStateUpdatedAt.Equal(newer.OccurredAt) {
		t.Fatalf("stale unlike changed persisted like state: %+v", state)
	}
	stats := readInteractionStats(t, db, videoID)
	if stats.LikeCount != 1 {
		t.Fatalf("stale unlike changed like count: %+v", stats)
	}
}

func TestVideoInteractionWriterIgnoresStaleLikeSetAfterNewerUnset(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	db.SeedUser(t, userID)
	db.SeedVideo(t, videoID, 100000)
	writer := catalogrepo.NewVideoInteractionWriter(db.Pool)

	if _, err := writer.SetVideoLike(context.Background(), likeCommandAt(true, time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC))); err != nil {
		t.Fatalf("initial like: %v", err)
	}
	newerUnset := likeCommandAt(false, time.Date(2026, 5, 23, 12, 10, 0, 0, time.UTC))
	if _, err := writer.SetVideoLike(context.Background(), newerUnset); err != nil {
		t.Fatalf("newer unlike: %v", err)
	}

	staleSet := likeCommandAt(true, time.Date(2026, 5, 23, 12, 5, 0, 0, time.UTC))
	result, err := writer.SetVideoLike(context.Background(), staleSet)
	if err != nil {
		t.Fatalf("stale like: %v", err)
	}
	if result.HasLiked || result.LikeCount != 0 {
		t.Fatalf("stale like should return current unliked state/count: %+v", result)
	}

	state := readInteractionState(t, db, userID, videoID)
	if state.HasLiked || state.LikedAt != nil || state.LikeStateUpdatedAt == nil || !state.LikeStateUpdatedAt.Equal(newerUnset.OccurredAt) {
		t.Fatalf("stale like changed persisted unliked state: %+v", state)
	}
}

func TestVideoInteractionWriterIgnoresStaleFavoriteStateChanges(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	db.SeedUser(t, userID)
	db.SeedVideo(t, videoID, 100000)
	writer := catalogrepo.NewVideoInteractionWriter(db.Pool)

	newer := favoriteCommandAt(true, time.Date(2026, 5, 23, 12, 10, 0, 0, time.UTC))
	if _, err := writer.SetVideoFavorite(context.Background(), newer); err != nil {
		t.Fatalf("newer favorite: %v", err)
	}

	stale := favoriteCommandAt(false, time.Date(2026, 5, 23, 12, 5, 0, 0, time.UTC))
	result, err := writer.SetVideoFavorite(context.Background(), stale)
	if err != nil {
		t.Fatalf("stale unfavorite: %v", err)
	}
	if !result.HasFavorited || result.FavoriteCount != 1 {
		t.Fatalf("stale unfavorite should return current favorite state/count: %+v", result)
	}

	state := readInteractionState(t, db, userID, videoID)
	if !state.HasBookmarked || state.BookmarkedAt == nil || !state.BookmarkedAt.Equal(newer.OccurredAt) || state.FavoriteStateUpdatedAt == nil || !state.FavoriteStateUpdatedAt.Equal(newer.OccurredAt) {
		t.Fatalf("stale unfavorite changed persisted favorite state: %+v", state)
	}
	stats := readInteractionStats(t, db, videoID)
	if stats.FavoriteCount != 1 {
		t.Fatalf("stale unfavorite changed favorite count: %+v", stats)
	}
}

func TestVideoInteractionWriterIgnoresStaleFavoriteSetAfterNewerUnset(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	db.SeedUser(t, userID)
	db.SeedVideo(t, videoID, 100000)
	writer := catalogrepo.NewVideoInteractionWriter(db.Pool)

	if _, err := writer.SetVideoFavorite(context.Background(), favoriteCommandAt(true, time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC))); err != nil {
		t.Fatalf("initial favorite: %v", err)
	}
	newerUnset := favoriteCommandAt(false, time.Date(2026, 5, 23, 12, 10, 0, 0, time.UTC))
	if _, err := writer.SetVideoFavorite(context.Background(), newerUnset); err != nil {
		t.Fatalf("newer unfavorite: %v", err)
	}

	staleSet := favoriteCommandAt(true, time.Date(2026, 5, 23, 12, 5, 0, 0, time.UTC))
	result, err := writer.SetVideoFavorite(context.Background(), staleSet)
	if err != nil {
		t.Fatalf("stale favorite: %v", err)
	}
	if result.HasFavorited || result.FavoriteCount != 0 {
		t.Fatalf("stale favorite should return current unfavorited state/count: %+v", result)
	}

	state := readInteractionState(t, db, userID, videoID)
	if state.HasBookmarked || state.BookmarkedAt != nil || state.FavoriteStateUpdatedAt == nil || !state.FavoriteStateUpdatedAt.Equal(newerUnset.OccurredAt) {
		t.Fatalf("stale favorite changed persisted unfavorited state: %+v", state)
	}
}

func TestVideoInteractionWriterDeleteDoesNotCreateUserState(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	db.SeedUser(t, userID)
	db.SeedVideo(t, videoID, 100000)
	writer := catalogrepo.NewVideoInteractionWriter(db.Pool)

	result, err := writer.SetVideoLike(context.Background(), likeCommand(false))
	if err != nil {
		t.Fatalf("unset like without state: %v", err)
	}
	if result.HasLiked || result.LikeCount != 0 {
		t.Fatalf("unexpected unset result: %+v", result)
	}
	if countInteractionStates(t, db, userID, videoID) != 0 {
		t.Fatal("delete no-op must not create an empty user state row")
	}
}

func TestVideoInteractionWriterRejectsUnavailableVideos(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, db *fixture.TestDatabase)
	}{
		{name: "missing", setup: func(t *testing.T, db *fixture.TestDatabase) {}},
		{name: "inactive", setup: func(t *testing.T, db *fixture.TestDatabase) {
			db.SeedVideo(t, videoID, 100000)
			updateVideoAvailability(t, db, "inactive", "public", nil)
		}},
		{name: "private", setup: func(t *testing.T, db *fixture.TestDatabase) {
			db.SeedVideo(t, videoID, 100000)
			updateVideoAvailability(t, db, "active", "private", nil)
		}},
		{name: "future", setup: func(t *testing.T, db *fixture.TestDatabase) {
			db.SeedVideo(t, videoID, 100000)
			future := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
			updateVideoAvailability(t, db, "active", "public", &future)
		}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			db := suite.CreateTestDatabase(t)
			db.SeedUser(t, userID)
			tt.setup(t, db)
			writer := catalogrepo.NewVideoInteractionWriter(db.Pool)

			_, err := writer.SetVideoFavorite(context.Background(), favoriteCommand(true))
			if !errors.Is(err, apprepo.ErrVideoNotFound) {
				t.Fatalf("expected ErrVideoNotFound, got %v", err)
			}
		})
	}
}

func likeCommand(enabled bool) model.VideoLikeCommand {
	return likeCommandAt(enabled, time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC))
}

func likeCommandAt(enabled bool, occurredAt time.Time) model.VideoLikeCommand {
	return model.VideoLikeCommand{
		UserID:     userID,
		VideoID:    videoID,
		Enabled:    enabled,
		OccurredAt: occurredAt,
	}
}

func favoriteCommand(enabled bool) model.VideoFavoriteCommand {
	return favoriteCommandAt(enabled, time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC))
}

func favoriteCommandAt(enabled bool, occurredAt time.Time) model.VideoFavoriteCommand {
	return model.VideoFavoriteCommand{
		UserID:     userID,
		VideoID:    videoID,
		Enabled:    enabled,
		OccurredAt: occurredAt,
	}
}

type interactionStateRow struct {
	HasLiked               bool
	HasBookmarked          bool
	LikedAt                *time.Time
	BookmarkedAt           *time.Time
	LikeStateUpdatedAt     *time.Time
	FavoriteStateUpdatedAt *time.Time
}

func readInteractionState(t *testing.T, db *fixture.TestDatabase, userID string, videoID string) interactionStateRow {
	t.Helper()
	var row interactionStateRow
	if err := db.Pool.QueryRow(context.Background(), `
		select has_liked, has_bookmarked, liked_at, bookmarked_at, like_state_updated_at, favorite_state_updated_at
		from catalog.video_user_states
		where user_id = $1 and video_id = $2`, userID, videoID).Scan(
		&row.HasLiked,
		&row.HasBookmarked,
		&row.LikedAt,
		&row.BookmarkedAt,
		&row.LikeStateUpdatedAt,
		&row.FavoriteStateUpdatedAt,
	); err != nil {
		t.Fatalf("read interaction state: %v", err)
	}
	return row
}

type interactionStatsRow struct {
	LikeCount     int64
	FavoriteCount int64
}

func readInteractionStats(t *testing.T, db *fixture.TestDatabase, videoID string) interactionStatsRow {
	t.Helper()
	var row interactionStatsRow
	if err := db.Pool.QueryRow(context.Background(), `
		select like_count, favorite_count
		from catalog.video_engagement_stats
		where video_id = $1`, videoID).Scan(
		&row.LikeCount,
		&row.FavoriteCount,
	); err != nil {
		t.Fatalf("read interaction stats: %v", err)
	}
	return row
}

func countInteractionStates(t *testing.T, db *fixture.TestDatabase, userID string, videoID string) int {
	t.Helper()
	var count int
	if err := db.Pool.QueryRow(context.Background(), `
		select count(*)
		from catalog.video_user_states
		where user_id = $1 and video_id = $2`, userID, videoID).Scan(&count); err != nil {
		t.Fatalf("count interaction states: %v", err)
	}
	return count
}

func updateVideoAvailability(t *testing.T, db *fixture.TestDatabase, status string, visibility string, publishAt *time.Time) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `
		update catalog.videos
		set status = $2, visibility_status = $3, publish_at = $4
		where video_id = $1`, videoID, status, visibility, publishAt); err != nil {
		t.Fatalf("update video availability: %v", err)
	}
}
