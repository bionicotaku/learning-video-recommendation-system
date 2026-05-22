package service_test

import (
	"context"
	"testing"
	"time"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
)

func TestVideoFavoritesServiceBuildsPublicResponse(t *testing.T) {
	favoritedAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	list := &fakeVideoFavoritesList{
		response: catalogdto.ListVideoFavoritesResponse{
			Items: []catalogdto.VideoFavoriteItem{{
				VideoID:       "11111111-1111-1111-1111-111111111111",
				Title:         "Favorite",
				CoverImageURL: stringPtr("covers/111.webp"),
				DurationMS:    90500,
				ViewCount:     12,
				FavoritedAt:   favoritedAt,
			}},
			Page: catalogdto.VideoLibraryPage{Limit: 20, HasMore: true, NextCursor: stringPtr("cursor-1")},
		},
	}
	service := apiservice.NewVideoLibraryService(list, &fakeVideoHistoryList{}, apiservice.NewPublicAssetURLBuilder("https://cdn.example.com/assets"))

	response, err := service.ListFavorites(context.Background(), apvdto.ListVideoFavoritesRequest{
		UserID: "user-1",
		Limit:  20,
		Cursor: "cursor-0",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if list.request.UserID != "user-1" || list.request.Limit != 20 || list.request.Cursor != "cursor-0" {
		t.Fatalf("list request = %+v", list.request)
	}
	if len(response.Items) != 1 || response.Items[0].CoverImageURL == nil || *response.Items[0].CoverImageURL != "https://cdn.example.com/assets/covers/111.webp" {
		t.Fatalf("items = %+v", response.Items)
	}
	if response.Items[0].DurationSeconds != 91 || response.Items[0].FavoritedAt != favoritedAt {
		t.Fatalf("item = %+v", response.Items[0])
	}
	if !response.Page.HasMore || response.Page.NextCursor == nil || *response.Page.NextCursor != "cursor-1" {
		t.Fatalf("page = %+v", response.Page)
	}
}

func TestVideoHistoryServiceBuildsPublicResponse(t *testing.T) {
	lastWatchedAt := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)
	list := &fakeVideoHistoryList{
		response: catalogdto.ListVideoHistoryResponse{
			Items: []catalogdto.VideoHistoryItem{{
				VideoID:        "22222222-2222-2222-2222-222222222222",
				Title:          "History",
				DurationMS:     61000,
				ViewCount:      7,
				LastPositionMS: 12000,
				LastWatchedAt:  lastWatchedAt,
			}},
			Page: catalogdto.VideoLibraryPage{Limit: 20},
		},
	}
	service := apiservice.NewVideoLibraryService(&fakeVideoFavoritesList{}, list, apiservice.NewPublicAssetURLBuilder("https://cdn.example.com/assets"))

	response, err := service.ListHistory(context.Background(), apvdto.ListVideoHistoryRequest{
		UserID: "user-1",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(response.Items) != 1 || response.Items[0].CoverImageURL != nil {
		t.Fatalf("items = %+v", response.Items)
	}
	if response.Items[0].DurationSeconds != 61 || response.Items[0].LastPositionMS != 12000 || response.Items[0].LastWatchedAt != lastWatchedAt {
		t.Fatalf("item = %+v", response.Items[0])
	}
}

func TestVideoLibraryServicesValidateUserID(t *testing.T) {
	service := apiservice.NewVideoLibraryService(&fakeVideoFavoritesList{}, &fakeVideoHistoryList{}, apiservice.NewPublicAssetURLBuilder("https://cdn.example.com/assets"))

	if _, err := service.ListFavorites(context.Background(), apvdto.ListVideoFavoritesRequest{}); err == nil || !apiservice.IsInvalidRequest(err) {
		t.Fatalf("favorites error = %v, want invalid request", err)
	}
	if _, err := service.ListHistory(context.Background(), apvdto.ListVideoHistoryRequest{}); err == nil || !apiservice.IsInvalidRequest(err) {
		t.Fatalf("history error = %v, want invalid request", err)
	}
}

type fakeVideoFavoritesList struct {
	request  catalogdto.ListVideoFavoritesRequest
	response catalogdto.ListVideoFavoritesResponse
	err      error
}

func (f *fakeVideoFavoritesList) Execute(ctx context.Context, request catalogdto.ListVideoFavoritesRequest) (catalogdto.ListVideoFavoritesResponse, error) {
	f.request = request
	if f.err != nil {
		return catalogdto.ListVideoFavoritesResponse{}, f.err
	}
	return f.response, nil
}

type fakeVideoHistoryList struct {
	request  catalogdto.ListVideoHistoryRequest
	response catalogdto.ListVideoHistoryResponse
	err      error
}

func (f *fakeVideoHistoryList) Execute(ctx context.Context, request catalogdto.ListVideoHistoryRequest) (catalogdto.ListVideoHistoryResponse, error) {
	f.request = request
	if f.err != nil {
		return catalogdto.ListVideoHistoryResponse{}, f.err
	}
	return f.response, nil
}
