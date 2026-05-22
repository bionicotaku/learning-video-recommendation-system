package service_test

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/catalog/application/dto"
	catalogservice "learning-video-recommendation-system/internal/catalog/application/service"
	"learning-video-recommendation-system/internal/catalog/domain/model"
)

func TestListVideoFavoritesUsecasePaginatesWithOpaqueCursor(t *testing.T) {
	now := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	reader := &fakeVideoLibraryReader{
		favorites: []model.VideoFavoriteListItem{
			{VideoID: "11111111-1111-1111-1111-111111111111", Title: "One", FavoritedAt: now},
			{VideoID: "22222222-2222-2222-2222-222222222222", Title: "Two", FavoritedAt: now.Add(-time.Minute)},
		},
	}
	usecase := catalogservice.NewListVideoFavoritesUsecase(reader)

	firstPage, err := usecase.Execute(context.Background(), dto.ListVideoFavoritesRequest{
		UserID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Limit:  1,
	})
	if err != nil {
		t.Fatalf("Execute() first page error = %v", err)
	}
	if len(firstPage.Items) != 1 || firstPage.Items[0].VideoID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("first page items = %+v", firstPage.Items)
	}
	if firstPage.Page.Limit != 1 || !firstPage.Page.HasMore || firstPage.Page.NextCursor == nil {
		t.Fatalf("page = %+v, want has_more next_cursor", firstPage.Page)
	}
	if len(reader.favoriteQueries) != 1 || reader.favoriteQueries[0].LimitPlusOne != 2 || reader.favoriteQueries[0].Cursor != nil {
		t.Fatalf("first query = %+v", reader.favoriteQueries)
	}

	_, err = usecase.Execute(context.Background(), dto.ListVideoFavoritesRequest{
		UserID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Limit:  1,
		Cursor: *firstPage.Page.NextCursor,
	})
	if err != nil {
		t.Fatalf("Execute() second page error = %v", err)
	}
	cursor := reader.favoriteQueries[1].Cursor
	if cursor == nil || cursor.Kind != dto.VideoLibraryCursorKindFavorites || cursor.VideoID != "11111111-1111-1111-1111-111111111111" || !cursor.SortAt.Equal(now) {
		t.Fatalf("decoded cursor = %+v", cursor)
	}
}

func TestListVideoHistoryUsecaseDefaultsAndRejectsWrongCursorKind(t *testing.T) {
	reader := &fakeVideoLibraryReader{}
	usecase := catalogservice.NewListVideoHistoryUsecase(reader)

	response, err := usecase.Execute(context.Background(), dto.ListVideoHistoryRequest{
		UserID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.Page.Limit != 20 || response.Page.HasMore || response.Page.NextCursor != nil || len(response.Items) != 0 {
		t.Fatalf("response = %+v, want default empty page", response)
	}

	wrongCursor := base64.RawURLEncoding.EncodeToString([]byte(`{"kind":"video_favorites","at":"2026-05-22T10:00:00Z","video_id":"11111111-1111-1111-1111-111111111111"}`))
	_, err = usecase.Execute(context.Background(), dto.ListVideoHistoryRequest{
		UserID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Cursor: wrongCursor,
	})
	if err == nil || !catalogservice.IsValidationError(err) {
		t.Fatalf("error = %v, want validation error", err)
	}
}

func TestListVideoLibraryUsecasesValidateInputs(t *testing.T) {
	favorites := catalogservice.NewListVideoFavoritesUsecase(&fakeVideoLibraryReader{})
	history := catalogservice.NewListVideoHistoryUsecase(&fakeVideoLibraryReader{})
	validUserID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	if _, err := favorites.Execute(context.Background(), dto.ListVideoFavoritesRequest{Limit: 20}); err == nil || !catalogservice.IsValidationError(err) {
		t.Fatalf("favorites missing user error = %v, want validation", err)
	}
	if _, err := history.Execute(context.Background(), dto.ListVideoHistoryRequest{UserID: validUserID, Limit: 101}); err == nil || !catalogservice.IsValidationError(err) {
		t.Fatalf("history invalid limit error = %v, want validation", err)
	}
	if _, err := favorites.Execute(context.Background(), dto.ListVideoFavoritesRequest{UserID: validUserID, Cursor: "not-base64"}); err == nil || !catalogservice.IsValidationError(err) {
		t.Fatalf("favorites malformed cursor error = %v, want validation", err)
	}
}

type fakeVideoLibraryReader struct {
	favoriteQueries []dto.ListVideoFavoritesQuery
	historyQueries  []dto.ListVideoHistoryQuery
	favorites       []model.VideoFavoriteListItem
	history         []model.VideoHistoryListItem
}

func (f *fakeVideoLibraryReader) ListVideoFavorites(ctx context.Context, query dto.ListVideoFavoritesQuery) ([]model.VideoFavoriteListItem, error) {
	f.favoriteQueries = append(f.favoriteQueries, query)
	return f.favorites, nil
}

func (f *fakeVideoLibraryReader) ListVideoHistory(ctx context.Context, query dto.ListVideoHistoryQuery) ([]model.VideoHistoryListItem, error) {
	f.historyQueries = append(f.historyQueries, query)
	return f.history, nil
}
