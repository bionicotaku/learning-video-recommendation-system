package videolibrary_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/videolibrary"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/router"
)

func TestVideoFavoritesReturnsListAndPassesPrincipal(t *testing.T) {
	service := &fakeFavoritesService{
		response: apvdto.ListVideoFavoritesResponse{
			Items: []apvdto.VideoFavoriteItem{{
				VideoID:         "11111111-1111-1111-1111-111111111111",
				Title:           "Favorite",
				CoverImageURL:   stringPtr("https://cdn.example.com/covers/111.webp"),
				DurationSeconds: 91,
				ViewCount:       12,
				FavoritedAt:     time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
			}},
			Page: apvdto.VideoLibraryPage{Limit: 20},
		},
	}
	server := newServer(service)
	t.Cleanup(server.Close)

	response := get(t, server, "/api/video-favorites?limit=20&cursor=cursor-1", true)
	requireStatus(t, response, http.StatusOK)
	var body map[string]any
	decodeJSON(t, response, &body)
	items := body["items"].([]any)
	item := items[0].(map[string]any)
	assertDetailFieldsAbsent(t, item)
	if service.request.UserID != "user-1" || service.request.Limit != 20 || service.request.Cursor != "cursor-1" {
		t.Fatalf("request = %+v", service.request)
	}
}

func TestVideoHistoryReturnsListAndPassesPrincipal(t *testing.T) {
	service := &fakeVideoLibraryService{
		historyResponse: apvdto.ListVideoHistoryResponse{
			Items: []apvdto.VideoHistoryItem{{
				VideoID:         "22222222-2222-2222-2222-222222222222",
				Title:           "History",
				CoverImageURL:   stringPtr("https://cdn.example.com/covers/222.webp"),
				DurationSeconds: 61,
				ViewCount:       7,
				LastPositionMS:  12000,
				LastWatchedAt:   time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
			}},
			Page: apvdto.VideoLibraryPage{Limit: 20},
		},
	}
	server := newServer(service)
	t.Cleanup(server.Close)

	response := get(t, server, "/api/video-history?limit=20&cursor=cursor-2", true)
	requireStatus(t, response, http.StatusOK)
	var body map[string]any
	decodeJSON(t, response, &body)
	items := body["items"].([]any)
	item := items[0].(map[string]any)
	assertDetailFieldsAbsent(t, item)
	if service.historyRequest.UserID != "user-1" || service.historyRequest.Limit != 20 || service.historyRequest.Cursor != "cursor-2" {
		t.Fatalf("request = %+v", service.historyRequest)
	}
}

func TestVideoHistoryReturnsEmptyList(t *testing.T) {
	server := newServer(&fakeHistoryService{
		response: apvdto.ListVideoHistoryResponse{
			Items: []apvdto.VideoHistoryItem{},
			Page:  apvdto.VideoLibraryPage{Limit: 20},
		},
	})
	t.Cleanup(server.Close)

	response := get(t, server, "/api/video-history", true)
	requireStatus(t, response, http.StatusOK)
	var body apvdto.ListVideoHistoryResponse
	decodeJSON(t, response, &body)
	if len(body.Items) != 0 || body.Page.Limit != 20 {
		t.Fatalf("body = %+v, want empty page", body)
	}
}

func TestVideoLibraryRejectsMissingPrincipalAndBadQuery(t *testing.T) {
	t.Run("missing principal", func(t *testing.T) {
		server := newServer(&fakeVideoLibraryService{})
		t.Cleanup(server.Close)

		response := get(t, server, "/api/video-favorites", false)
		requireStatus(t, response, http.StatusUnauthorized)
	})

	t.Run("invalid limit", func(t *testing.T) {
		server := newServer(&fakeVideoLibraryService{})
		t.Cleanup(server.Close)

		response := get(t, server, "/api/video-history?limit=0", true)
		requireStatus(t, response, http.StatusBadRequest)
	})

	t.Run("limit too high", func(t *testing.T) {
		server := newServer(&fakeVideoLibraryService{})
		t.Cleanup(server.Close)

		response := get(t, server, "/api/video-favorites?limit=101", true)
		requireStatus(t, response, http.StatusBadRequest)
	})

	t.Run("malformed cursor", func(t *testing.T) {
		server := newServer(&fakeVideoLibraryService{favoritesErr: apiservice.InvalidRequestError("cursor is invalid")})
		t.Cleanup(server.Close)

		response := get(t, server, "/api/video-favorites?cursor=not-base64", true)
		requireStatus(t, response, http.StatusBadRequest)
	})

	t.Run("wrong cursor kind", func(t *testing.T) {
		server := newServer(&fakeVideoLibraryService{historyErr: apiservice.InvalidRequestError("cursor kind does not match endpoint")})
		t.Cleanup(server.Close)

		response := get(t, server, "/api/video-history?cursor=wrong-kind", true)
		requireStatus(t, response, http.StatusBadRequest)
	})
}

func newServer(service videolibrary.VideoLibraryService) *httptest.Server {
	handler := router.New(router.Options{
		VideoLibrary: videolibrary.NewHandler(service),
	})
	handler = auth.PrincipalMiddleware(auth.Options{GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo"})(handler)
	handler = middleware.RequestID(handler)
	return httptest.NewServer(handler)
}

func get(t *testing.T, server *httptest.Server, path string, withPrincipal bool) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if withPrincipal {
		request.Header.Set("X-Apigateway-Api-Userinfo", "eyJzdWIiOiJ1c2VyLTEifQ")
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	return response
}

func requireStatus(t *testing.T, response *http.Response, want int) {
	t.Helper()
	if response.StatusCode != want {
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		t.Fatalf("status = %d, want %d: %s", response.StatusCode, want, string(body))
	}
}

func decodeJSON(t *testing.T, response *http.Response, target any) {
	t.Helper()
	defer response.Body.Close()
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

func stringPtr(value string) *string {
	return &value
}

func assertDetailFieldsAbsent(t *testing.T, item map[string]any) {
	t.Helper()
	for _, field := range []string{"video_url", "transcript_url", "description", "like_count", "favorite_count", "has_liked", "has_favorited", "user_state"} {
		if _, ok := item[field]; ok {
			t.Fatalf("item leaked detail field %q: %+v", field, item)
		}
	}
}

type fakeVideoLibraryService struct {
	favoritesRequest  apvdto.ListVideoFavoritesRequest
	historyRequest    apvdto.ListVideoHistoryRequest
	favoritesResponse apvdto.ListVideoFavoritesResponse
	historyResponse   apvdto.ListVideoHistoryResponse
	favoritesErr      error
	historyErr        error
}

func (f *fakeVideoLibraryService) ListFavorites(ctx context.Context, request apvdto.ListVideoFavoritesRequest) (apvdto.ListVideoFavoritesResponse, error) {
	f.favoritesRequest = request
	if f.favoritesErr != nil {
		return apvdto.ListVideoFavoritesResponse{}, f.favoritesErr
	}
	return f.favoritesResponse, nil
}

func (f *fakeVideoLibraryService) ListHistory(ctx context.Context, request apvdto.ListVideoHistoryRequest) (apvdto.ListVideoHistoryResponse, error) {
	f.historyRequest = request
	if f.historyErr != nil {
		return apvdto.ListVideoHistoryResponse{}, f.historyErr
	}
	return f.historyResponse, nil
}

type fakeFavoritesService struct {
	request  apvdto.ListVideoFavoritesRequest
	response apvdto.ListVideoFavoritesResponse
	err      error
}

func (f *fakeFavoritesService) ListFavorites(ctx context.Context, request apvdto.ListVideoFavoritesRequest) (apvdto.ListVideoFavoritesResponse, error) {
	f.request = request
	if f.err != nil {
		return apvdto.ListVideoFavoritesResponse{}, f.err
	}
	return f.response, nil
}

func (f *fakeFavoritesService) ListHistory(ctx context.Context, request apvdto.ListVideoHistoryRequest) (apvdto.ListVideoHistoryResponse, error) {
	return apvdto.ListVideoHistoryResponse{}, nil
}

type fakeHistoryService struct {
	request  apvdto.ListVideoHistoryRequest
	response apvdto.ListVideoHistoryResponse
	err      error
}

func (f *fakeHistoryService) ListFavorites(ctx context.Context, request apvdto.ListVideoFavoritesRequest) (apvdto.ListVideoFavoritesResponse, error) {
	return apvdto.ListVideoFavoritesResponse{}, nil
}

func (f *fakeHistoryService) ListHistory(ctx context.Context, request apvdto.ListVideoHistoryRequest) (apvdto.ListVideoHistoryResponse, error) {
	f.request = request
	if f.err != nil {
		return apvdto.ListVideoHistoryResponse{}, f.err
	}
	return f.response, nil
}
