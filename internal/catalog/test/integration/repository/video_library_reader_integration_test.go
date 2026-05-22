//go:build integration

package repository_test

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/catalog/application/dto"
	catalogservice "learning-video-recommendation-system/internal/catalog/application/service"
	catalogrepo "learning-video-recommendation-system/internal/catalog/infrastructure/persistence/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestVideoLibraryReaderListVideoFavoritesPaginatesAndFilters(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()

	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	firstID := "11111111-1111-1111-1111-111111111111"
	secondID := "22222222-2222-2222-2222-222222222222"
	olderID := "33333333-3333-3333-3333-333333333333"
	dirtyID := "44444444-4444-4444-4444-444444444444"
	inactiveID := "55555555-5555-5555-5555-555555555555"
	privateID := "66666666-6666-6666-6666-666666666666"
	futureID := "77777777-7777-7777-7777-777777777777"

	now := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	older := now.Add(-time.Hour)
	future := now.Add(24 * time.Hour)

	seedVideoLibraryUser(t, db.Pool, userID)
	seedFeedVideo(t, db.Pool, firstID, "First", "", "hls/first.m3u8", "covers/first.webp", "active", "public", nil)
	seedFeedVideo(t, db.Pool, secondID, "Second", "", "hls/second.m3u8", "", "active", "public", nil)
	seedFeedVideo(t, db.Pool, olderID, "Older", "", "hls/older.m3u8", "", "active", "public", nil)
	seedFeedVideo(t, db.Pool, dirtyID, "Dirty", "", "hls/dirty.m3u8", "", "active", "public", nil)
	seedFeedVideo(t, db.Pool, inactiveID, "Inactive", "", "hls/inactive.m3u8", "", "inactive", "public", nil)
	seedFeedVideo(t, db.Pool, privateID, "Private", "", "hls/private.m3u8", "", "active", "private", nil)
	seedFeedVideo(t, db.Pool, futureID, "Future", "", "hls/future.m3u8", "", "active", "public", &future)
	seedVideoStats(t, db.Pool, firstID, 12)

	if _, err := db.Pool.Exec(ctx, `
		insert into catalog.video_user_states (user_id, video_id, has_bookmarked, bookmarked_at)
		values
			($1, $2, true, $5),
			($1, $3, true, $5),
			($1, $4, true, $6),
			($1, $7, true, null),
			($1, $8, true, $5),
			($1, $9, true, $5),
			($1, $10, true, $5)
	`, userID, firstID, secondID, olderID, now, older, dirtyID, inactiveID, privateID, futureID); err != nil {
		t.Fatalf("seed favorite states: %v", err)
	}

	usecase := catalogservice.NewListVideoFavoritesUsecase(catalogrepo.NewVideoLibraryReader(db.Pool))
	firstPage, err := usecase.Execute(ctx, dto.ListVideoFavoritesRequest{UserID: userID, Limit: 2})
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if got := videoFavoriteIDs(firstPage.Items); len(got) != 2 || got[0] != firstID || got[1] != secondID {
		t.Fatalf("first page ids = %+v", got)
	}
	if firstPage.Items[0].CoverImageURL == nil || *firstPage.Items[0].CoverImageURL != "covers/first.webp" || firstPage.Items[0].ViewCount != 12 {
		t.Fatalf("first item = %+v", firstPage.Items[0])
	}
	if firstPage.Items[1].ViewCount != 0 {
		t.Fatalf("missing stats should default view_count=0: %+v", firstPage.Items[1])
	}
	if !firstPage.Page.HasMore || firstPage.Page.NextCursor == nil {
		t.Fatalf("first page page = %+v, want next cursor", firstPage.Page)
	}

	secondPage, err := usecase.Execute(ctx, dto.ListVideoFavoritesRequest{
		UserID: userID,
		Limit:  2,
		Cursor: *firstPage.Page.NextCursor,
	})
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if got := videoFavoriteIDs(secondPage.Items); len(got) != 1 || got[0] != olderID {
		t.Fatalf("second page ids = %+v, want older only", got)
	}
	if secondPage.Page.HasMore || secondPage.Page.NextCursor != nil {
		t.Fatalf("second page page = %+v, want terminal page", secondPage.Page)
	}
}

func TestVideoLibraryReaderListVideoHistoryPaginatesAndFilters(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()

	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	firstID := "11111111-1111-1111-1111-111111111111"
	secondID := "22222222-2222-2222-2222-222222222222"
	olderID := "33333333-3333-3333-3333-333333333333"
	dirtyID := "44444444-4444-4444-4444-444444444444"
	privateID := "55555555-5555-5555-5555-555555555555"

	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	older := now.Add(-time.Hour)

	seedVideoLibraryUser(t, db.Pool, userID)
	seedFeedVideo(t, db.Pool, firstID, "First history", "", "hls/first-history.m3u8", "covers/first-history.webp", "active", "public", nil)
	seedFeedVideo(t, db.Pool, secondID, "Second history", "", "hls/second-history.m3u8", "", "active", "public", nil)
	seedFeedVideo(t, db.Pool, olderID, "Older history", "", "hls/older-history.m3u8", "", "active", "public", nil)
	seedFeedVideo(t, db.Pool, dirtyID, "Dirty history", "", "hls/dirty-history.m3u8", "", "active", "public", nil)
	seedFeedVideo(t, db.Pool, privateID, "Private history", "", "hls/private-history.m3u8", "", "active", "private", nil)
	seedVideoStats(t, db.Pool, firstID, 5)

	if _, err := db.Pool.Exec(ctx, `
		insert into catalog.video_user_states (
			user_id,
			video_id,
			has_watched,
			first_watched_at,
			last_watched_at,
			watch_count,
			last_position_ms,
			max_position_ms,
			total_watch_ms
		) values
			($1, $2, true, $6, $6, 1, 12000, 12000, 12000),
			($1, $3, true, $6, $6, 1, 8000, 8000, 8000),
			($1, $4, true, $7, $7, 1, 3000, 3000, 3000),
			($1, $5, true, $6, null, 1, 9000, 9000, 9000),
			($1, $8, true, $6, $6, 1, 4000, 4000, 4000)
	`, userID, firstID, secondID, olderID, dirtyID, now, older, privateID); err != nil {
		t.Fatalf("seed history states: %v", err)
	}

	usecase := catalogservice.NewListVideoHistoryUsecase(catalogrepo.NewVideoLibraryReader(db.Pool))
	firstPage, err := usecase.Execute(ctx, dto.ListVideoHistoryRequest{UserID: userID, Limit: 2})
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if got := videoHistoryIDs(firstPage.Items); len(got) != 2 || got[0] != firstID || got[1] != secondID {
		t.Fatalf("first page ids = %+v", got)
	}
	if firstPage.Items[0].LastPositionMS != 12000 || firstPage.Items[0].ViewCount != 5 {
		t.Fatalf("first history item = %+v", firstPage.Items[0])
	}
	if firstPage.Page.NextCursor == nil {
		t.Fatalf("first page page = %+v, want next cursor", firstPage.Page)
	}

	secondPage, err := usecase.Execute(ctx, dto.ListVideoHistoryRequest{
		UserID: userID,
		Limit:  2,
		Cursor: *firstPage.Page.NextCursor,
	})
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if got := videoHistoryIDs(secondPage.Items); len(got) != 1 || got[0] != olderID {
		t.Fatalf("second page ids = %+v, want older only", got)
	}
}

func TestVideoLibraryUsecaseRejectsInvalidCursorsBeforeRepositoryRead(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()

	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	usecaseFavorites := catalogservice.NewListVideoFavoritesUsecase(catalogrepo.NewVideoLibraryReader(db.Pool))
	usecaseHistory := catalogservice.NewListVideoHistoryUsecase(catalogrepo.NewVideoLibraryReader(db.Pool))

	if _, err := usecaseFavorites.Execute(ctx, dto.ListVideoFavoritesRequest{
		UserID: userID,
		Cursor: "not-base64",
	}); err == nil || !catalogservice.IsValidationError(err) {
		t.Fatalf("malformed cursor error = %v, want validation", err)
	}

	favoriteCursor := base64.RawURLEncoding.EncodeToString([]byte(`{"kind":"video_favorites","at":"2026-05-22T10:00:00Z","video_id":"11111111-1111-1111-1111-111111111111"}`))
	if _, err := usecaseHistory.Execute(ctx, dto.ListVideoHistoryRequest{
		UserID: userID,
		Cursor: favoriteCursor,
	}); err == nil || !catalogservice.IsValidationError(err) {
		t.Fatalf("wrong-kind cursor error = %v, want validation", err)
	}
}

func videoFavoriteIDs(items []dto.VideoFavoriteItem) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.VideoID)
	}
	return result
}

func videoHistoryIDs(items []dto.VideoHistoryItem) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.VideoID)
	}
	return result
}

func seedVideoLibraryUser(t *testing.T, pool *pgxpool.Pool, userID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		insert into auth.users (id, email)
		values ($1::uuid, $1::text || '@example.com')
		on conflict (id) do nothing`, userID); err != nil {
		t.Fatalf("seed auth user: %v", err)
	}
}

func seedVideoStats(t *testing.T, pool *pgxpool.Pool, videoID string, viewCount int64) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		insert into catalog.video_engagement_stats (video_id, view_count, like_count, favorite_count)
		values ($1, $2, 0, 0)`, videoID, viewCount); err != nil {
		t.Fatalf("seed video stats: %v", err)
	}
}
