package videointeractions_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/videointeractions"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/router"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
	catalogservice "learning-video-recommendation-system/internal/catalog/application/service"
)

func TestVideoLikeRoutesMapRequestAndReturnLikeOnly(t *testing.T) {
	like := &fakeLikeUsecase{
		response: catalogdto.VideoLikeResponse{
			VideoID:   videoID,
			HasLiked:  true,
			LikeCount: 86,
		},
	}
	server := newServer(like, &fakeFavoriteUsecase{})
	t.Cleanup(server.Close)

	response := requestInteraction(t, server, http.MethodPut, "/api/videos/"+videoID+"/like", true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	var body map[string]any
	decodeJSON(t, response, &body)
	if body["video_id"] != videoID || body["has_liked"] != true || body["like_count"].(float64) != 86 {
		t.Fatalf("unexpected response: %+v", body)
	}
	if _, ok := body["has_favorited"]; ok {
		t.Fatalf("like response must not include favorite state: %+v", body)
	}
	if _, ok := body["favorite_count"]; ok {
		t.Fatalf("like response must not include favorite count: %+v", body)
	}
	if !like.called || like.request.UserID != userID || like.request.VideoID != videoID || !like.request.Enabled {
		t.Fatalf("unexpected like request: %+v", like.request)
	}
}

func TestVideoLikeDeleteMapsUnset(t *testing.T) {
	like := &fakeLikeUsecase{
		response: catalogdto.VideoLikeResponse{
			VideoID:   videoID,
			HasLiked:  false,
			LikeCount: 85,
		},
	}
	server := newServer(like, &fakeFavoriteUsecase{})
	t.Cleanup(server.Close)

	response := requestInteraction(t, server, http.MethodDelete, "/api/videos/"+videoID+"/like", true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	var body catalogdto.VideoLikeResponse
	decodeJSON(t, response, &body)
	if body.HasLiked || body.LikeCount != 85 {
		t.Fatalf("unexpected response: %+v", body)
	}
	if !like.called || like.request.Enabled {
		t.Fatalf("expected unset request, got %+v", like.request)
	}
}

func TestVideoFavoriteRoutesMapRequestAndReturnFavoriteOnly(t *testing.T) {
	favorite := &fakeFavoriteUsecase{
		response: catalogdto.VideoFavoriteResponse{
			VideoID:       videoID,
			HasFavorited:  true,
			FavoriteCount: 17,
		},
	}
	server := newServer(&fakeLikeUsecase{}, favorite)
	t.Cleanup(server.Close)

	response := requestInteraction(t, server, http.MethodPut, "/api/videos/"+videoID+"/favorite", true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	var body map[string]any
	decodeJSON(t, response, &body)
	if body["video_id"] != videoID || body["has_favorited"] != true || body["favorite_count"].(float64) != 17 {
		t.Fatalf("unexpected response: %+v", body)
	}
	if _, ok := body["has_liked"]; ok {
		t.Fatalf("favorite response must not include like state: %+v", body)
	}
	if _, ok := body["like_count"]; ok {
		t.Fatalf("favorite response must not include like count: %+v", body)
	}
	if !favorite.called || favorite.request.UserID != userID || favorite.request.VideoID != videoID || !favorite.request.Enabled {
		t.Fatalf("unexpected favorite request: %+v", favorite.request)
	}
}

func TestVideoInteractionsRejectInvalidVideoIDAndMissingPrincipal(t *testing.T) {
	t.Run("invalid video id", func(t *testing.T) {
		like := &fakeLikeUsecase{}
		server := newServer(like, &fakeFavoriteUsecase{})
		t.Cleanup(server.Close)

		response := requestInteraction(t, server, http.MethodPut, "/api/videos/not-a-uuid/like", true)
		if response.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
		}
		if like.called {
			t.Fatal("like service should not be called")
		}
	})

	t.Run("missing principal", func(t *testing.T) {
		like := &fakeLikeUsecase{}
		server := newServer(like, &fakeFavoriteUsecase{})
		t.Cleanup(server.Close)

		response := requestInteraction(t, server, http.MethodPut, "/api/videos/"+videoID+"/like", false)
		if response.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", response.StatusCode, readBody(t, response))
		}
		if like.called {
			t.Fatal("like service should not be called")
		}
	})
}

func TestVideoInteractionsMapCatalogErrors(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "catalog validation", err: catalogservice.UnprocessableError("bad payload"), status: http.StatusUnprocessableEntity, code: "unprocessable_entity"},
		{name: "not found", err: catalogservice.NotFoundError("video not found"), status: http.StatusNotFound, code: "not_found"},
		{name: "internal", err: errors.New("db down"), status: http.StatusInternalServerError, code: "internal_error"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			server := newServer(&fakeLikeUsecase{err: tt.err}, &fakeFavoriteUsecase{})
			t.Cleanup(server.Close)

			response := requestInteraction(t, server, http.MethodPut, "/api/videos/"+videoID+"/like", true)
			if response.StatusCode != tt.status {
				t.Fatalf("expected %d, got %d: %s", tt.status, response.StatusCode, readBody(t, response))
			}
			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			decodeJSON(t, response, &body)
			if body.Error.Code != tt.code {
				t.Fatalf("expected code %q, got %q", tt.code, body.Error.Code)
			}
		})
	}
}

func newServer(like *fakeLikeUsecase, favorite *fakeFavoriteUsecase) *httptest.Server {
	group := videointeractions.NewHandler(like, favorite)
	handler := router.New(router.Options{VideoInteractions: group})
	handler = auth.TrustedHeaderPrincipalMiddleware("X-Trusted-User-ID")(handler)
	handler = middleware.RequestID(handler)
	return httptest.NewServer(handler)
}

func requestInteraction(t *testing.T, server *httptest.Server, method string, path string, withPrincipal bool) *http.Response {
	t.Helper()
	request, err := http.NewRequest(method, server.URL+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if withPrincipal {
		request.Header.Set("X-Trusted-User-ID", userID)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	return response
}

func decodeJSON(t *testing.T, response *http.Response, target any) {
	t.Helper()
	defer response.Body.Close()
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

func readBody(t *testing.T, response *http.Response) string {
	t.Helper()
	defer response.Body.Close()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(response.Body)
	return buf.String()
}

type fakeLikeUsecase struct {
	called   bool
	request  catalogdto.SetVideoLikeRequest
	response catalogdto.VideoLikeResponse
	err      error
}

func (f *fakeLikeUsecase) Execute(ctx context.Context, request catalogdto.SetVideoLikeRequest) (catalogdto.VideoLikeResponse, error) {
	f.called = true
	f.request = request
	return f.response, f.err
}

type fakeFavoriteUsecase struct {
	called   bool
	request  catalogdto.SetVideoFavoriteRequest
	response catalogdto.VideoFavoriteResponse
	err      error
}

func (f *fakeFavoriteUsecase) Execute(ctx context.Context, request catalogdto.SetVideoFavoriteRequest) (catalogdto.VideoFavoriteResponse, error) {
	f.called = true
	f.request = request
	return f.response, f.err
}

const (
	userID  = "11111111-1111-1111-1111-111111111111"
	videoID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
)
