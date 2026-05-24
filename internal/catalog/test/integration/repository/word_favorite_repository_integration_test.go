//go:build integration

package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/catalog/application/dto"
	catalogservice "learning-video-recommendation-system/internal/catalog/application/service"
	catalogrepo "learning-video-recommendation-system/internal/catalog/infrastructure/persistence/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestWordFavoriteRepositorySupportsCoarseAndTokenStatus(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	videoID := "00000000-0000-4000-8000-000000000001"
	db.SeedUser(t, userID)
	seedWordFavoriteCoarseUnit(t, db.Pool, 108404, "make", "verb", "制作", "制作；做；使成为。", "active")
	seedWordFavoriteVideoToken(t, db.Pool, videoID, 7, 2, "making", "正在制作")

	repository := catalogrepo.NewWordFavoriteRepository(db.Pool)
	set := catalogservice.NewSetWordFavoriteUsecase(repository)
	status := catalogservice.NewGetWordFavoriteStatusUsecase(repository)

	if err := set.Execute(ctx, dto.SetWordFavoriteRequest{
		UserID:        userID,
		CoarseUnitID:  int64Ptr(108404),
		Text:          "Making",
		Source:        dto.WordFavoriteSourceVideoTranscript,
		VideoID:       stringPtr(videoID),
		SentenceIndex: int32Ptr(7),
		TokenIndex:    int32Ptr(2),
		OccurredAt:    wordFavoriteTime(2),
	}); err != nil {
		t.Fatalf("set coarse favorite from transcript: %v", err)
	}

	response, err := status.Execute(ctx, dto.GetWordFavoriteStatusRequest{
		UserID:        userID,
		CoarseUnitID:  int64Ptr(108404),
		Text:          "Making",
		Source:        dto.WordFavoriteSourceVideoTranscript,
		VideoID:       stringPtr(videoID),
		SentenceIndex: int32Ptr(7),
		TokenIndex:    int32Ptr(2),
	})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !response.IsFavorited {
		t.Fatal("expected coarse-key transcript status to be favorited")
	}
}

func TestWordFavoriteListPaginatesAndProjectsDisplayFields(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	firstVideoID := "00000000-0000-4000-8000-000000000001"
	secondVideoID := "00000000-0000-4000-8000-000000000002"
	db.SeedUser(t, userID)
	seedWordFavoriteCoarseUnit(t, db.Pool, 108404, "make", "verb", "制作", "制作；做；使成为。", "active")
	seedWordFavoriteCoarseUnit(t, db.Pool, 108405, "carry", "verb", "携带", "携带；传达。", "active")
	seedWordFavoriteVideoToken(t, db.Pool, firstVideoID, 7, 2, "making", "正在制作")
	seedWordFavoriteVideoToken(t, db.Pool, secondVideoID, 4, 1, "carry weight", "有分量")

	repository := catalogrepo.NewWordFavoriteRepository(db.Pool)
	set := catalogservice.NewSetWordFavoriteUsecase(repository)
	list := catalogservice.NewListWordFavoritesUsecase(repository)

	if err := set.Execute(ctx, dto.SetWordFavoriteRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList, OccurredAt: wordFavoriteTime(1)}); err != nil {
		t.Fatalf("set first: %v", err)
	}
	if err := set.Execute(ctx, dto.SetWordFavoriteRequest{UserID: userID, CoarseUnitID: nil, Text: "carry weight", Source: dto.WordFavoriteSourceVideoTranscript, VideoID: stringPtr(secondVideoID), SentenceIndex: int32Ptr(4), TokenIndex: int32Ptr(1), OccurredAt: wordFavoriteTime(2)}); err != nil {
		t.Fatalf("set second: %v", err)
	}

	firstPage, err := list.Execute(ctx, dto.ListWordFavoritesRequest{UserID: userID, Limit: 1})
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if len(firstPage.Items) != 1 || firstPage.Items[0].Source != dto.WordFavoriteSourceVideoTranscript || firstPage.Items[0].SourceText == nil || *firstPage.Items[0].SourceText != "carry weight" {
		t.Fatalf("first page item = %+v", firstPage.Items)
	}
	if firstPage.Items[0].SourceTranslation == nil || *firstPage.Items[0].SourceTranslation != "取得进步需要练习。" {
		t.Fatalf("source translation = %+v, want sentence translation", firstPage.Items[0].SourceTranslation)
	}
	if !firstPage.Page.HasMore || firstPage.Page.NextCursor == nil {
		t.Fatalf("first page = %+v, want next cursor", firstPage.Page)
	}

	secondPage, err := list.Execute(ctx, dto.ListWordFavoritesRequest{UserID: userID, Limit: 1, Cursor: *firstPage.Page.NextCursor})
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if len(secondPage.Items) != 1 || secondPage.Items[0].Label == nil || *secondPage.Items[0].Label != "make" || secondPage.Items[0].ChineseLabel == nil || *secondPage.Items[0].ChineseLabel != "制作" {
		t.Fatalf("second page item = %+v", secondPage.Items)
	}
}

func TestWordFavoriteCoarseTranscriptFavoriteSurvivesSourceVideoDelete(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	videoID := "00000000-0000-4000-8000-000000000001"
	db.SeedUser(t, userID)
	seedWordFavoriteCoarseUnit(t, db.Pool, 108404, "make", "verb", "制作", "制作；做；使成为。", "active")
	seedWordFavoriteVideoToken(t, db.Pool, videoID, 7, 2, "making", "正在制作")

	repository := catalogrepo.NewWordFavoriteRepository(db.Pool)
	set := catalogservice.NewSetWordFavoriteUsecase(repository)
	status := catalogservice.NewGetWordFavoriteStatusUsecase(repository)

	if err := set.Execute(ctx, dto.SetWordFavoriteRequest{
		UserID:        userID,
		CoarseUnitID:  int64Ptr(108404),
		Text:          "making",
		Source:        dto.WordFavoriteSourceVideoTranscript,
		VideoID:       stringPtr(videoID),
		SentenceIndex: int32Ptr(7),
		TokenIndex:    int32Ptr(2),
		OccurredAt:    wordFavoriteTime(1),
	}); err != nil {
		t.Fatalf("set: %v", err)
	}
	if _, err := db.Pool.Exec(ctx, `delete from catalog.videos where video_id = $1::uuid`, videoID); err != nil {
		t.Fatalf("delete source video: %v", err)
	}

	response, err := status.Execute(ctx, dto.GetWordFavoriteStatusRequest{
		UserID:        userID,
		CoarseUnitID:  int64Ptr(108404),
		Text:          "making",
		Source:        dto.WordFavoriteSourceVideoTranscript,
		VideoID:       stringPtr(videoID),
		SentenceIndex: int32Ptr(7),
		TokenIndex:    int32Ptr(2),
	})
	if err != nil {
		t.Fatalf("status after source delete: %v", err)
	}
	if !response.IsFavorited {
		t.Fatal("coarse-key favorite should survive source video deletion")
	}
}

func TestWordFavoriteListFiltersTokenOnlyWhenVideoBecomesHidden(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	videoID := "00000000-0000-4000-8000-000000000001"
	db.SeedUser(t, userID)
	seedWordFavoriteVideoToken(t, db.Pool, videoID, 7, 2, "making", "正在制作")

	repository := catalogrepo.NewWordFavoriteRepository(db.Pool)
	set := catalogservice.NewSetWordFavoriteUsecase(repository)
	list := catalogservice.NewListWordFavoritesUsecase(repository)

	if err := set.Execute(ctx, dto.SetWordFavoriteRequest{
		UserID:        userID,
		Text:          "making",
		Source:        dto.WordFavoriteSourceVideoTranscript,
		VideoID:       stringPtr(videoID),
		SentenceIndex: int32Ptr(7),
		TokenIndex:    int32Ptr(2),
		OccurredAt:    wordFavoriteTime(1),
	}); err != nil {
		t.Fatalf("set token favorite: %v", err)
	}
	if _, err := db.Pool.Exec(ctx, `update catalog.videos set visibility_status = 'private' where video_id = $1::uuid`, videoID); err != nil {
		t.Fatalf("hide video: %v", err)
	}

	response, err := list.Execute(ctx, dto.ListWordFavoritesRequest{UserID: userID})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(response.Items) != 0 {
		t.Fatalf("items = %+v, want hidden token-only favorite filtered", response.Items)
	}
}

func TestWordFavoriteDuplicateSetKeepsFavoritedAt(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	db.SeedUser(t, userID)
	seedWordFavoriteCoarseUnit(t, db.Pool, 108404, "make", "verb", "制作", "制作；做；使成为。", "active")

	repository := catalogrepo.NewWordFavoriteRepository(db.Pool)
	set := catalogservice.NewSetWordFavoriteUsecase(repository)

	request := dto.SetWordFavoriteRequest{
		UserID:       userID,
		CoarseUnitID: int64Ptr(108404),
		Text:         "make",
		Source:       dto.WordFavoriteSourceWordList,
		OccurredAt:   wordFavoriteTime(1),
	}
	if err := set.Execute(ctx, request); err != nil {
		t.Fatalf("first set: %v", err)
	}
	original := request.OccurredAt
	newerRequest := request
	newerRequest.OccurredAt = wordFavoriteTime(3)
	if err := set.Execute(ctx, newerRequest); err != nil {
		t.Fatalf("duplicate set: %v", err)
	}

	var got time.Time
	var stateUpdatedAt time.Time
	if err := db.Pool.QueryRow(ctx, `
		select favorited_at, state_updated_at
		from catalog.word_favorites
		where user_id = $1::uuid and coarse_unit_id = $2
	`, userID, 108404).Scan(&got, &stateUpdatedAt); err != nil {
		t.Fatalf("query favorited_at: %v", err)
	}
	if !got.Equal(original) {
		t.Fatalf("favorited_at = %s, want %s", got, original)
	}
	if !stateUpdatedAt.Equal(newerRequest.OccurredAt) {
		t.Fatalf("state_updated_at = %s, want %s", stateUpdatedAt, newerRequest.OccurredAt)
	}
	if gotCount := countWordFavoriteRows(t, db.Pool, userID, 108404); gotCount != 1 {
		t.Fatalf("row count = %d, want 1", gotCount)
	}
}

func TestWordFavoriteStaleDeleteDoesNotUnsetNewerSet(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	db.SeedUser(t, userID)
	seedWordFavoriteCoarseUnit(t, db.Pool, 108404, "make", "verb", "制作", "制作；做；使成为。", "active")

	repository := catalogrepo.NewWordFavoriteRepository(db.Pool)
	set := catalogservice.NewSetWordFavoriteUsecase(repository)
	unset := catalogservice.NewUnsetWordFavoriteUsecase(repository)
	status := catalogservice.NewGetWordFavoriteStatusUsecase(repository)
	list := catalogservice.NewListWordFavoritesUsecase(repository)

	if err := set.Execute(ctx, dto.SetWordFavoriteRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList, OccurredAt: wordFavoriteTime(2)}); err != nil {
		t.Fatalf("set t2: %v", err)
	}
	if err := unset.Execute(ctx, dto.UnsetWordFavoriteRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList, OccurredAt: wordFavoriteTime(1)}); err != nil {
		t.Fatalf("stale unset t1: %v", err)
	}

	response, err := status.Execute(ctx, dto.GetWordFavoriteStatusRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !response.IsFavorited {
		t.Fatal("stale delete should not remove newer favorite")
	}
	page, err := list.Execute(ctx, dto.ListWordFavoritesRequest{UserID: userID})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("items = %+v, want favorite visible", page.Items)
	}
}

func TestWordFavoriteStaleSetDoesNotRestoreNewerDelete(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	db.SeedUser(t, userID)
	seedWordFavoriteCoarseUnit(t, db.Pool, 108404, "make", "verb", "制作", "制作；做；使成为。", "active")

	repository := catalogrepo.NewWordFavoriteRepository(db.Pool)
	set := catalogservice.NewSetWordFavoriteUsecase(repository)
	unset := catalogservice.NewUnsetWordFavoriteUsecase(repository)
	status := catalogservice.NewGetWordFavoriteStatusUsecase(repository)
	list := catalogservice.NewListWordFavoritesUsecase(repository)

	if err := unset.Execute(ctx, dto.UnsetWordFavoriteRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList, OccurredAt: wordFavoriteTime(2)}); err != nil {
		t.Fatalf("unset t2: %v", err)
	}
	if err := set.Execute(ctx, dto.SetWordFavoriteRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList, OccurredAt: wordFavoriteTime(1)}); err != nil {
		t.Fatalf("stale set t1: %v", err)
	}

	response, err := status.Execute(ctx, dto.GetWordFavoriteStatusRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if response.IsFavorited {
		t.Fatal("stale set should not restore newer delete")
	}
	page, err := list.Execute(ctx, dto.ListWordFavoritesRequest{UserID: userID})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("items = %+v, want tombstone hidden", page.Items)
	}
	state := readWordFavoriteState(t, db.Pool, userID, 108404)
	if state.IsFavorited || state.FavoritedAt.Valid || !state.StateUpdatedAt.Equal(wordFavoriteTime(2)) {
		t.Fatalf("state = %+v, want t2 tombstone", state)
	}
}

func TestWordFavoriteTombstoneBlocksOldPutAndNewerPutRestores(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	db.SeedUser(t, userID)
	seedWordFavoriteCoarseUnit(t, db.Pool, 108404, "make", "verb", "制作", "制作；做；使成为。", "active")

	repository := catalogrepo.NewWordFavoriteRepository(db.Pool)
	set := catalogservice.NewSetWordFavoriteUsecase(repository)
	unset := catalogservice.NewUnsetWordFavoriteUsecase(repository)
	status := catalogservice.NewGetWordFavoriteStatusUsecase(repository)
	list := catalogservice.NewListWordFavoritesUsecase(repository)

	if err := unset.Execute(ctx, dto.UnsetWordFavoriteRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList, OccurredAt: wordFavoriteTime(2)}); err != nil {
		t.Fatalf("unset t2: %v", err)
	}
	if gotCount := countWordFavoriteRows(t, db.Pool, userID, 108404); gotCount != 1 {
		t.Fatalf("row count after tombstone = %d, want 1", gotCount)
	}
	if err := set.Execute(ctx, dto.SetWordFavoriteRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList, OccurredAt: wordFavoriteTime(1)}); err != nil {
		t.Fatalf("stale set t1: %v", err)
	}
	response, err := status.Execute(ctx, dto.GetWordFavoriteStatusRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList})
	if err != nil {
		t.Fatalf("status after stale set: %v", err)
	}
	if response.IsFavorited {
		t.Fatal("old put should be blocked by tombstone")
	}
	if err := set.Execute(ctx, dto.SetWordFavoriteRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList, OccurredAt: wordFavoriteTime(3)}); err != nil {
		t.Fatalf("restore set t3: %v", err)
	}

	response, err = status.Execute(ctx, dto.GetWordFavoriteStatusRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList})
	if err != nil {
		t.Fatalf("status after restore: %v", err)
	}
	if !response.IsFavorited {
		t.Fatal("newer put should restore favorite")
	}
	page, err := list.Execute(ctx, dto.ListWordFavoritesRequest{UserID: userID})
	if err != nil {
		t.Fatalf("list after restore: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("items = %+v, want restored favorite visible", page.Items)
	}
	state := readWordFavoriteState(t, db.Pool, userID, 108404)
	if !state.IsFavorited || !state.FavoritedAt.Valid || !state.FavoritedAt.Time.Equal(wordFavoriteTime(3)) || !state.StateUpdatedAt.Equal(wordFavoriteTime(3)) {
		t.Fatalf("state = %+v, want restored t3", state)
	}
}

func TestWordFavoriteStaleCoarseSetSkipsTargetValidation(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	db.SeedUser(t, userID)
	seedWordFavoriteCoarseUnit(t, db.Pool, 108404, "make", "verb", "制作", "制作；做；使成为。", "active")

	repository := catalogrepo.NewWordFavoriteRepository(db.Pool)
	set := catalogservice.NewSetWordFavoriteUsecase(repository)
	unset := catalogservice.NewUnsetWordFavoriteUsecase(repository)
	status := catalogservice.NewGetWordFavoriteStatusUsecase(repository)

	if err := unset.Execute(ctx, dto.UnsetWordFavoriteRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList, OccurredAt: wordFavoriteTime(2)}); err != nil {
		t.Fatalf("unset t2: %v", err)
	}
	if _, err := db.Pool.Exec(ctx, `update semantic.coarse_unit set status = 'inactive' where id = $1`, 108404); err != nil {
		t.Fatalf("deactivate coarse unit: %v", err)
	}
	if err := set.Execute(ctx, dto.SetWordFavoriteRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList, OccurredAt: wordFavoriteTime(1)}); err != nil {
		t.Fatalf("stale set should skip target validation, got: %v", err)
	}
	response, err := status.Execute(ctx, dto.GetWordFavoriteStatusRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if response.IsFavorited {
		t.Fatal("stale set should not restore tombstone")
	}
}

func TestWordFavoriteDuplicateCoarseSetSkipsTargetValidationAfterContentChange(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	db.SeedUser(t, userID)
	seedWordFavoriteCoarseUnit(t, db.Pool, 108404, "make", "verb", "制作", "制作；做；使成为。", "active")

	repository := catalogrepo.NewWordFavoriteRepository(db.Pool)
	set := catalogservice.NewSetWordFavoriteUsecase(repository)
	status := catalogservice.NewGetWordFavoriteStatusUsecase(repository)
	occurredAt := wordFavoriteTime(1)

	request := dto.SetWordFavoriteRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList, OccurredAt: occurredAt}
	if err := set.Execute(ctx, request); err != nil {
		t.Fatalf("set t1: %v", err)
	}
	if _, err := db.Pool.Exec(ctx, `update semantic.coarse_unit set status = 'inactive' where id = $1`, 108404); err != nil {
		t.Fatalf("deactivate coarse unit: %v", err)
	}
	if err := set.Execute(ctx, request); err != nil {
		t.Fatalf("duplicate set should skip target validation, got: %v", err)
	}
	response, err := status.Execute(ctx, dto.GetWordFavoriteStatusRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !response.IsFavorited {
		t.Fatal("duplicate set should keep favorite state")
	}
}

func TestWordFavoriteCoarseDeleteWritesTombstoneWhenTargetMissing(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	db.SeedUser(t, userID)

	repository := catalogrepo.NewWordFavoriteRepository(db.Pool)
	set := catalogservice.NewSetWordFavoriteUsecase(repository)
	unset := catalogservice.NewUnsetWordFavoriteUsecase(repository)
	status := catalogservice.NewGetWordFavoriteStatusUsecase(repository)

	if err := unset.Execute(ctx, dto.UnsetWordFavoriteRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList, OccurredAt: wordFavoriteTime(2)}); err != nil {
		t.Fatalf("unset missing coarse target: %v", err)
	}
	if gotCount := countWordFavoriteRows(t, db.Pool, userID, 108404); gotCount != 1 {
		t.Fatalf("row count after missing-target tombstone = %d, want 1", gotCount)
	}
	if err := set.Execute(ctx, dto.SetWordFavoriteRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList, OccurredAt: wordFavoriteTime(1)}); err != nil {
		t.Fatalf("stale set should be blocked before target validation, got: %v", err)
	}
	response, err := status.Execute(ctx, dto.GetWordFavoriteStatusRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if response.IsFavorited {
		t.Fatal("missing-target tombstone should block old put")
	}
}

func TestWordFavoriteNonStaleCoarseSetStillRequiresActiveTarget(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	db.SeedUser(t, userID)
	seedWordFavoriteCoarseUnit(t, db.Pool, 108404, "make", "verb", "制作", "制作；做；使成为。", "inactive")

	repository := catalogrepo.NewWordFavoriteRepository(db.Pool)
	set := catalogservice.NewSetWordFavoriteUsecase(repository)

	err := set.Execute(ctx, dto.SetWordFavoriteRequest{UserID: userID, CoarseUnitID: int64Ptr(108404), Text: "make", Source: dto.WordFavoriteSourceWordList, OccurredAt: wordFavoriteTime(3)})
	if err == nil || !catalogservice.IsNotFoundError(err) {
		t.Fatalf("set inactive coarse error = %v, want not found", err)
	}
}

func TestWordFavoriteStaleTokenSetSkipsTargetValidation(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	videoID := "00000000-0000-4000-8000-000000000001"
	db.SeedUser(t, userID)
	seedWordFavoriteVideoToken(t, db.Pool, videoID, 7, 2, "making", "正在制作")

	repository := catalogrepo.NewWordFavoriteRepository(db.Pool)
	set := catalogservice.NewSetWordFavoriteUsecase(repository)
	unset := catalogservice.NewUnsetWordFavoriteUsecase(repository)
	status := catalogservice.NewGetWordFavoriteStatusUsecase(repository)

	if err := unset.Execute(ctx, dto.UnsetWordFavoriteRequest{UserID: userID, Text: "making", Source: dto.WordFavoriteSourceVideoTranscript, VideoID: stringPtr(videoID), SentenceIndex: int32Ptr(7), TokenIndex: int32Ptr(2), OccurredAt: wordFavoriteTime(2)}); err != nil {
		t.Fatalf("unset t2: %v", err)
	}
	if _, err := db.Pool.Exec(ctx, `update catalog.videos set visibility_status = 'private' where video_id = $1::uuid`, videoID); err != nil {
		t.Fatalf("hide video: %v", err)
	}
	if err := set.Execute(ctx, dto.SetWordFavoriteRequest{UserID: userID, Text: "making", Source: dto.WordFavoriteSourceVideoTranscript, VideoID: stringPtr(videoID), SentenceIndex: int32Ptr(7), TokenIndex: int32Ptr(2), OccurredAt: wordFavoriteTime(1)}); err != nil {
		t.Fatalf("stale set should skip token validation, got: %v", err)
	}
	response, err := status.Execute(ctx, dto.GetWordFavoriteStatusRequest{UserID: userID, Text: "making", Source: dto.WordFavoriteSourceVideoTranscript, VideoID: stringPtr(videoID), SentenceIndex: int32Ptr(7), TokenIndex: int32Ptr(2)})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if response.IsFavorited {
		t.Fatal("stale set should not restore token tombstone")
	}
}

func TestWordFavoriteDuplicateTokenSetSkipsTargetValidationAfterContentChange(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	videoID := "00000000-0000-4000-8000-000000000001"
	db.SeedUser(t, userID)
	seedWordFavoriteVideoToken(t, db.Pool, videoID, 7, 2, "making", "正在制作")

	repository := catalogrepo.NewWordFavoriteRepository(db.Pool)
	set := catalogservice.NewSetWordFavoriteUsecase(repository)
	status := catalogservice.NewGetWordFavoriteStatusUsecase(repository)
	occurredAt := wordFavoriteTime(1)

	request := dto.SetWordFavoriteRequest{UserID: userID, Text: "making", Source: dto.WordFavoriteSourceVideoTranscript, VideoID: stringPtr(videoID), SentenceIndex: int32Ptr(7), TokenIndex: int32Ptr(2), OccurredAt: occurredAt}
	if err := set.Execute(ctx, request); err != nil {
		t.Fatalf("set t1: %v", err)
	}
	if _, err := db.Pool.Exec(ctx, `update catalog.videos set visibility_status = 'private' where video_id = $1::uuid`, videoID); err != nil {
		t.Fatalf("hide video: %v", err)
	}
	if err := set.Execute(ctx, request); err != nil {
		t.Fatalf("duplicate set should skip token validation, got: %v", err)
	}
	response, err := status.Execute(ctx, dto.GetWordFavoriteStatusRequest{UserID: userID, Text: "making", Source: dto.WordFavoriteSourceVideoTranscript, VideoID: stringPtr(videoID), SentenceIndex: int32Ptr(7), TokenIndex: int32Ptr(2)})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !response.IsFavorited {
		t.Fatal("duplicate set should keep token favorite state")
	}
}

func TestWordFavoriteNonStaleTokenSetStillRequiresVisibleTarget(t *testing.T) {
	db := suite.CreateTestDatabase(t)
	ctx := context.Background()
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	videoID := "00000000-0000-4000-8000-000000000001"
	db.SeedUser(t, userID)
	seedWordFavoriteVideoToken(t, db.Pool, videoID, 7, 2, "making", "正在制作")
	if _, err := db.Pool.Exec(ctx, `update catalog.videos set visibility_status = 'private' where video_id = $1::uuid`, videoID); err != nil {
		t.Fatalf("hide video: %v", err)
	}

	repository := catalogrepo.NewWordFavoriteRepository(db.Pool)
	set := catalogservice.NewSetWordFavoriteUsecase(repository)

	err := set.Execute(ctx, dto.SetWordFavoriteRequest{UserID: userID, Text: "making", Source: dto.WordFavoriteSourceVideoTranscript, VideoID: stringPtr(videoID), SentenceIndex: int32Ptr(7), TokenIndex: int32Ptr(2), OccurredAt: wordFavoriteTime(3)})
	if err == nil || !catalogservice.IsNotFoundError(err) {
		t.Fatalf("set hidden token error = %v, want not found", err)
	}
}

type wordFavoriteState struct {
	IsFavorited    bool
	FavoritedAt    sql.NullTime
	StateUpdatedAt time.Time
}

func readWordFavoriteState(t *testing.T, pool *pgxpool.Pool, userID string, coarseUnitID int64) wordFavoriteState {
	t.Helper()
	var state wordFavoriteState
	if err := pool.QueryRow(context.Background(), `
		select is_favorited, favorited_at, state_updated_at
		from catalog.word_favorites
		where user_id = $1::uuid and coarse_unit_id = $2
	`, userID, coarseUnitID).Scan(&state.IsFavorited, &state.FavoritedAt, &state.StateUpdatedAt); err != nil {
		t.Fatalf("query word favorite state: %v", err)
	}
	return state
}

func countWordFavoriteRows(t *testing.T, pool *pgxpool.Pool, userID string, coarseUnitID int64) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), `
		select count(*)
		from catalog.word_favorites
		where user_id = $1::uuid and coarse_unit_id = $2
	`, userID, coarseUnitID).Scan(&count); err != nil {
		t.Fatalf("count word favorite rows: %v", err)
	}
	return count
}

func seedWordFavoriteCoarseUnit(t *testing.T, pool *pgxpool.Pool, id int64, label string, pos string, chineseLabel string, chineseDef string, status string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		insert into semantic.coarse_unit (id, kind, label, pos, chinese_label, chinese_def, status)
		values ($1, 'word', $2, $3, $4, $5, $6)
	`, id, label, pos, chineseLabel, chineseDef, status); err != nil {
		t.Fatalf("seed coarse unit: %v", err)
	}
}

func seedWordFavoriteVideoToken(t *testing.T, pool *pgxpool.Pool, videoID string, sentenceIndex int32, tokenIndex int32, surfaceText string, translation string) {
	t.Helper()
	seedFeedVideo(t, pool, videoID, "Practice Makes Progress", "", "hls/"+videoID+".m3u8", "", "active", "public", nil)
	if _, err := pool.Exec(context.Background(), `
		insert into catalog.video_transcript_sentences (video_id, sentence_index, start_ms, end_ms, text, translation)
		values ($1::uuid, $2, 980, 2800, 'Making progress takes practice.', '取得进步需要练习。')
	`, videoID, sentenceIndex); err != nil {
		t.Fatalf("seed sentence: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		insert into catalog.video_semantic_spans (
			video_id,
			sentence_index,
			span_index,
			start_ms,
			end_ms,
			coarse_unit_id,
			surface_text,
			explanation,
			base_form,
			translation,
			dictionary
		) values (
			$1::uuid,
			$2,
			$3,
			1000,
			1200,
			null,
			$4,
			'Used in context.',
			$4,
			$5,
			'context dictionary'
		)
	`, videoID, sentenceIndex, tokenIndex, surfaceText, translation); err != nil {
		t.Fatalf("seed span: %v", err)
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}

func int32Ptr(value int32) *int32 {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func wordFavoriteTime(offsetSeconds int) time.Time {
	return time.Date(2026, 5, 24, 10, 20, 30+offsetSeconds, 123000000, time.UTC)
}
