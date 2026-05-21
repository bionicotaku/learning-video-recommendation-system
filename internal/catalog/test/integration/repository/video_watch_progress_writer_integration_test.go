//go:build integration

package repository_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	apprepo "learning-video-recommendation-system/internal/catalog/application/repository"
	"learning-video-recommendation-system/internal/catalog/domain/model"
	catalogrepo "learning-video-recommendation-system/internal/catalog/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/catalog/test/fixture"
	userapprepo "learning-video-recommendation-system/internal/user/application/repository"
	userrepo "learning-video-recommendation-system/internal/user/infrastructure/persistence/repository"

	"github.com/jackc/pgx/v5"
)

var suite *fixture.Suite

func TestMain(m *testing.M) {
	var err error
	suite, err = fixture.OpenSuite()
	if err != nil {
		panic(err)
	}
	code := m.Run()
	if err := suite.Close(); err != nil {
		panic(err)
	}
	os.Exit(code)
}

func TestVideoWatchProgressCreatesAndUpdatesProjections(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	db.SeedUser(t, userID)
	db.SeedVideo(t, videoID, 100000)
	writer := catalogrepo.NewVideoWatchProgressWriter(db.Pool)

	first, err := writer.RecordVideoWatchProgress(context.Background(), progressRequest(watchSessionID, 10000, 8000, at(0)))
	if err != nil {
		t.Fatalf("first record: %v", err)
	}
	if !first.CreatedSession || first.CompletedSession || first.DeltaActiveWatchMS != 8000 {
		t.Fatalf("unexpected first result: %+v", first)
	}

	second, err := writer.RecordVideoWatchProgress(context.Background(), progressRequest(watchSessionID, 5000, 7000, at(1)))
	if err != nil {
		t.Fatalf("second record: %v", err)
	}
	if second.CreatedSession || second.CompletedSession || second.DeltaActiveWatchMS != 0 {
		t.Fatalf("unexpected second result: %+v", second)
	}

	third, err := writer.RecordVideoWatchProgress(context.Background(), progressRequest(watchSessionID, 92000, 12000, at(2)))
	if err != nil {
		t.Fatalf("third record: %v", err)
	}
	if third.CreatedSession || !third.CompletedSession || third.DeltaActiveWatchMS != 4000 {
		t.Fatalf("unexpected third result: %+v", third)
	}

	userState := readUserState(t, db)
	if userState.WatchCount != 1 || userState.CompletedCount != 1 || userState.LastPositionMS != 92000 || userState.MaxPositionMS != 92000 || userState.TotalWatchMS != 12000 {
		t.Fatalf("unexpected user state: %+v", userState)
	}

	stats := readStats(t, db)
	if stats.ViewCount != 1 || stats.CompletedCount != 1 || stats.TotalWatchMS != 12000 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestVideoWatchProgressUpdatesUserActivityStatsInSameTransaction(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	db.SeedUser(t, userID)
	db.SeedVideo(t, videoID, 100000)
	writer := catalogrepo.NewVideoWatchProgressWriter(db.Pool, catalogrepo.WithWatchProgressActivityStats(func(tx pgx.Tx) userapprepo.ActivityStatsRecorder {
		return userrepo.NewRepository(tx)
	}))

	if _, err := writer.RecordVideoWatchProgress(context.Background(), progressRequest(watchSessionID, 10000, 8000, at(0))); err != nil {
		t.Fatalf("first record: %v", err)
	}
	if _, err := writer.RecordVideoWatchProgress(context.Background(), progressRequest(watchSessionID, 12000, 12000, at(1))); err != nil {
		t.Fatalf("second record: %v", err)
	}

	var totalWatchMS int64
	if err := db.Pool.QueryRow(context.Background(), `
		select total_watch_ms
		from app_user.user_activity_stats
		where user_id = $1`, userID).Scan(&totalWatchMS); err != nil {
		t.Fatalf("read user activity stats: %v", err)
	}
	if totalWatchMS != 12000 {
		t.Fatalf("total_watch_ms = %d, want 12000", totalWatchMS)
	}
	var dailyWatchMS int64
	if err := db.Pool.QueryRow(context.Background(), `
		select watch_ms
		from app_user.user_daily_activity_stats
		where user_id = $1`, userID).Scan(&dailyWatchMS); err != nil {
		t.Fatalf("read daily activity stats: %v", err)
	}
	if dailyWatchMS != 12000 {
		t.Fatalf("daily watch_ms = %d, want 12000", dailyWatchMS)
	}
}

func TestVideoWatchProgressDoesNotRollbackLastPositionForOldRetry(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	db.SeedUser(t, userID)
	db.SeedVideo(t, videoID, 100000)
	writer := catalogrepo.NewVideoWatchProgressWriter(db.Pool)

	if _, err := writer.RecordVideoWatchProgress(context.Background(), progressRequest(watchSessionID, 60000, 11000, at(10))); err != nil {
		t.Fatalf("record newer progress: %v", err)
	}
	if _, err := writer.RecordVideoWatchProgress(context.Background(), progressRequest(watchSessionID, 1000, 9000, at(1))); err != nil {
		t.Fatalf("record old retry: %v", err)
	}

	userState := readUserState(t, db)
	if userState.LastPositionMS != 60000 || userState.MaxPositionMS != 60000 || userState.TotalWatchMS != 11000 {
		t.Fatalf("old retry should not roll back state: %+v", userState)
	}
}

func TestVideoWatchProgressDoesNotRollbackLatestUserPositionForOlderSession(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	db.SeedUser(t, userID)
	db.SeedVideo(t, videoID, 100000)
	writer := catalogrepo.NewVideoWatchProgressWriter(db.Pool)

	if _, err := writer.RecordVideoWatchProgress(context.Background(), progressRequest(watchSessionID, 60000, 11000, at(10))); err != nil {
		t.Fatalf("record newer session: %v", err)
	}

	older := progressRequest("44444444-4444-4444-4444-444444444444", 1000, 6000, at(1))
	if _, err := writer.RecordVideoWatchProgress(context.Background(), older); err != nil {
		t.Fatalf("record older session: %v", err)
	}

	userState := readUserState(t, db)
	if userState.WatchCount != 2 || userState.LastPositionMS != 60000 || userState.MaxPositionMS != 60000 || userState.TotalWatchMS != 17000 {
		t.Fatalf("older session should not roll back latest user state: %+v", userState)
	}
	if !userState.LastWatchedAt.Equal(at(10)) {
		t.Fatalf("last_watched_at = %s, want %s", userState.LastWatchedAt, at(10))
	}
}

func TestVideoWatchProgressRejectsConflictingSessionAndMissingVideo(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	db.SeedUser(t, userID)
	db.SeedUser(t, otherUserID)
	db.SeedVideo(t, videoID, 100000)
	writer := catalogrepo.NewVideoWatchProgressWriter(db.Pool)

	if _, err := writer.RecordVideoWatchProgress(context.Background(), progressRequest(watchSessionID, 1000, 1000, at(0))); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	conflict := progressRequest(watchSessionID, 2000, 2000, at(1))
	conflict.UserID = otherUserID
	_, err := writer.RecordVideoWatchProgress(context.Background(), conflict)
	if !errors.Is(err, apprepo.ErrWatchSessionConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}

	missing := progressRequest("55555555-5555-5555-5555-555555555555", 1000, 1000, at(0))
	missing.VideoID = "99999999-9999-9999-9999-999999999999"
	_, err = writer.RecordVideoWatchProgress(context.Background(), missing)
	if !errors.Is(err, apprepo.ErrVideoNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestVideoWatchProgressConcurrentDuplicateSessionDoesNotDoubleCount(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	db.SeedUser(t, userID)
	db.SeedVideo(t, videoID, 100000)
	writer := catalogrepo.NewVideoWatchProgressWriter(db.Pool)

	positions := []int32{10000, 11000, 12000, 13000, 14000, 15000, 16000, 17000}
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, len(positions))
	for _, position := range positions {
		wg.Add(1)
		go func(position int32) {
			defer wg.Done()
			<-start
			_, err := writer.RecordVideoWatchProgress(context.Background(), progressRequest(watchSessionID, position, int64(position), at(int(position/10000))))
			errs <- err
		}(position)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent record: %v", err)
		}
	}

	userState := readUserState(t, db)
	if userState.WatchCount != 1 || userState.TotalWatchMS != 17000 || userState.MaxPositionMS != 17000 {
		t.Fatalf("expected single watch count, got %+v", userState)
	}
	stats := readStats(t, db)
	if stats.ViewCount != 1 || stats.TotalWatchMS != 17000 {
		t.Fatalf("expected single view count, got %+v", stats)
	}
}

func progressRequest(sessionID string, positionMS int32, activeWatchMS int64, occurredAt time.Time) model.VideoWatchProgress {
	return model.VideoWatchProgress{
		UserID:         userID,
		VideoID:        videoID,
		WatchSessionID: sessionID,
		PositionMS:     positionMS,
		ActiveWatchMS:  activeWatchMS,
		OccurredAt:     occurredAt,
		SourceSurface:  "fullscreen",
		ClientContext:  []byte(`{"platform":"ios"}`),
		Metadata:       []byte(`{"player":"native"}`),
	}
}

func at(offsetMinutes int) time.Time {
	return time.Date(2026, 5, 16, 12, offsetMinutes, 0, 0, time.UTC)
}

type userStateRow struct {
	WatchCount     int32
	CompletedCount int32
	LastWatchedAt  time.Time
	LastPositionMS int32
	MaxPositionMS  int32
	TotalWatchMS   int64
}

func readUserState(t *testing.T, db *fixture.TestDatabase) userStateRow {
	t.Helper()
	var row userStateRow
	if err := db.Pool.QueryRow(context.Background(), `
			select watch_count, completed_count, last_watched_at, last_position_ms, max_position_ms, total_watch_ms
			from catalog.video_user_states
			where user_id = $1 and video_id = $2`, userID, videoID).Scan(
		&row.WatchCount,
		&row.CompletedCount,
		&row.LastWatchedAt,
		&row.LastPositionMS,
		&row.MaxPositionMS,
		&row.TotalWatchMS,
	); err != nil {
		t.Fatalf("read user state: %v", err)
	}
	return row
}

type statsRow struct {
	ViewCount      int64
	CompletedCount int64
	TotalWatchMS   int64
}

func readStats(t *testing.T, db *fixture.TestDatabase) statsRow {
	t.Helper()
	var row statsRow
	if err := db.Pool.QueryRow(context.Background(), `
		select view_count, completed_count, total_watch_ms
		from catalog.video_engagement_stats
		where video_id = $1`, videoID).Scan(
		&row.ViewCount,
		&row.CompletedCount,
		&row.TotalWatchMS,
	); err != nil {
		t.Fatalf("read stats: %v", err)
	}
	return row
}

const (
	userID         = "11111111-1111-1111-1111-111111111111"
	otherUserID    = "22222222-2222-2222-2222-222222222222"
	videoID        = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	watchSessionID = "33333333-3333-3333-3333-333333333333"
)
