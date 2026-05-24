package wordfavorites_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/wordfavorites"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/router"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
	catalogservice "learning-video-recommendation-system/internal/catalog/application/service"
)

func TestWordFavoriteStatusMapsFlatIdentityAndReturnsContext(t *testing.T) {
	status := &fakeStatusUsecase{
		response: catalogdto.WordFavoriteStatusResponse{
			IsFavorited: true,
			VideoContext: &catalogdto.WordFavoriteVideoContext{
				VideoID:             wordVideoID,
				VideoTitle:          "Practice Makes Progress",
				VideoDurationMS:     int32Ptr(120000),
				SentenceIndex:       7,
				TokenIndex:          2,
				SentenceText:        "Making progress takes practice.",
				SentenceTranslation: stringPtr("取得进步需要练习。"),
				SentenceStartMS:     int32Ptr(980),
				SentenceEndMS:       int32Ptr(2800),
			},
		},
	}
	server := newServer(status, &fakeSetUsecase{}, &fakeUnsetUsecase{}, &fakeListUsecase{})
	t.Cleanup(server.Close)

	response := postJSON(t, server, "/api/word-favorites/status", `{
		"coarse_unit_id":108404,
		"text":"Making",
		"source":"video_transcript",
		"video_id":"`+wordVideoID+`",
		"sentence_index":7,
		"token_index":2,
		"include_video_context":true
	}`)
	requireStatus(t, response, http.StatusOK)

	if status.request.UserID != userID || status.request.CoarseUnitID == nil || *status.request.CoarseUnitID != 108404 || !status.request.IncludeVideoCtx {
		t.Fatalf("status request = %+v", status.request)
	}
	var body catalogdto.WordFavoriteStatusResponse
	decodeJSON(t, response, &body)
	if !body.IsFavorited || body.VideoContext == nil || body.VideoContext.VideoID != wordVideoID || body.VideoContext.SentenceIndex != 7 || body.VideoContext.TokenIndex != 2 {
		t.Fatalf("body = %+v", body)
	}
}

func TestWordFavoriteSetAndUnsetReturnNoContent(t *testing.T) {
	set := &fakeSetUsecase{}
	unset := &fakeUnsetUsecase{}
	server := newServer(&fakeStatusUsecase{}, set, unset, &fakeListUsecase{})
	t.Cleanup(server.Close)

	setResponse := putJSON(t, server, "/api/word-favorites", `{"coarse_unit_id":108404,"text":"Making","source":"word_list","occurred_at":"2026-05-24T10:20:30Z"}`)
	requireStatus(t, setResponse, http.StatusNoContent)
	requireEmptyBody(t, setResponse)
	if set.request.UserID != userID || set.request.Source != catalogdto.WordFavoriteSourceWordList {
		t.Fatalf("set request = %+v", set.request)
	}
	if !set.request.OccurredAt.Equal(time.Date(2026, 5, 24, 10, 20, 30, 0, time.UTC)) {
		t.Fatalf("set occurred_at = %s", set.request.OccurredAt)
	}

	unsetResponse := deleteJSON(t, server, "/api/word-favorites", `{"coarse_unit_id":108404,"text":"Making","source":"word_list","occurred_at":"2026-05-24T10:20:31Z"}`)
	requireStatus(t, unsetResponse, http.StatusNoContent)
	requireEmptyBody(t, unsetResponse)
	if unset.request.UserID != userID || unset.request.Source != catalogdto.WordFavoriteSourceWordList {
		t.Fatalf("unset request = %+v", unset.request)
	}
	if !unset.request.OccurredAt.Equal(time.Date(2026, 5, 24, 10, 20, 31, 0, time.UTC)) {
		t.Fatalf("unset occurred_at = %s", unset.request.OccurredAt)
	}
}

func TestWordFavoriteSetAndUnsetRequireOccurredAt(t *testing.T) {
	t.Run("put missing", func(t *testing.T) {
		set := &fakeSetUsecase{}
		server := newServer(&fakeStatusUsecase{}, set, &fakeUnsetUsecase{}, &fakeListUsecase{})
		t.Cleanup(server.Close)

		response := putJSON(t, server, "/api/word-favorites", `{"coarse_unit_id":108404,"text":"Making","source":"word_list"}`)
		requireStatus(t, response, http.StatusBadRequest)
		if set.called {
			t.Fatal("set usecase should not be called")
		}
	})

	t.Run("put invalid", func(t *testing.T) {
		set := &fakeSetUsecase{}
		server := newServer(&fakeStatusUsecase{}, set, &fakeUnsetUsecase{}, &fakeListUsecase{})
		t.Cleanup(server.Close)

		response := putJSON(t, server, "/api/word-favorites", `{"coarse_unit_id":108404,"text":"Making","source":"word_list","occurred_at":"yesterday"}`)
		requireStatus(t, response, http.StatusBadRequest)
		if set.called {
			t.Fatal("set usecase should not be called")
		}
	})

	t.Run("delete missing", func(t *testing.T) {
		unset := &fakeUnsetUsecase{}
		server := newServer(&fakeStatusUsecase{}, &fakeSetUsecase{}, unset, &fakeListUsecase{})
		t.Cleanup(server.Close)

		response := deleteJSON(t, server, "/api/word-favorites", `{"coarse_unit_id":108404,"text":"Making","source":"word_list"}`)
		requireStatus(t, response, http.StatusBadRequest)
		if unset.called {
			t.Fatal("unset usecase should not be called")
		}
	})

	t.Run("delete invalid", func(t *testing.T) {
		unset := &fakeUnsetUsecase{}
		server := newServer(&fakeStatusUsecase{}, &fakeSetUsecase{}, unset, &fakeListUsecase{})
		t.Cleanup(server.Close)

		response := deleteJSON(t, server, "/api/word-favorites", `{"coarse_unit_id":108404,"text":"Making","source":"word_list","occurred_at":"2026-05-24 10:20:30"}`)
		requireStatus(t, response, http.StatusBadRequest)
		if unset.called {
			t.Fatal("unset usecase should not be called")
		}
	})
}

func TestWordFavoriteListDefaultsToLimit50AndRejectsBadQuery(t *testing.T) {
	list := &fakeListUsecase{
		response: catalogdto.WordFavoriteListPage{
			Items: []catalogdto.WordFavoriteListItem{{
				CoarseUnitID:      int64Ptr(108404),
				Label:             stringPtr("make"),
				Pos:               stringPtr("verb"),
				ChineseLabel:      stringPtr("制作"),
				ChineseDef:        stringPtr("制作；做；使成为。"),
				Source:            catalogdto.WordFavoriteSourceWordList,
				VideoID:           nil,
				SentenceIndex:     nil,
				TokenIndex:        nil,
				SourceText:        nil,
				SourceTranslation: nil,
				SourceDictionary:  nil,
				SourceExplanation: nil,
			}},
			Page: catalogdto.WordFavoritePage{Limit: 50},
		},
	}
	server := newServer(&fakeStatusUsecase{}, &fakeSetUsecase{}, &fakeUnsetUsecase{}, list)
	t.Cleanup(server.Close)

	response := get(t, server, "/api/word-favorites")
	requireStatus(t, response, http.StatusOK)
	if list.request.UserID != userID || list.request.Limit != 0 || list.request.Cursor != "" {
		t.Fatalf("list request = %+v", list.request)
	}

	badLimit := get(t, server, "/api/word-favorites?limit=0")
	requireStatus(t, badLimit, http.StatusBadRequest)
}

func TestWordFavoriteHandlerRejectsUnknownFieldAndMapsCatalogNotFound(t *testing.T) {
	t.Run("unknown field", func(t *testing.T) {
		set := &fakeSetUsecase{}
		server := newServer(&fakeStatusUsecase{}, set, &fakeUnsetUsecase{}, &fakeListUsecase{})
		t.Cleanup(server.Close)

		response := putJSON(t, server, "/api/word-favorites", `{"coarse_unit_id":108404,"text":"Making","source":"word_list","occurred_at":"2026-05-24T10:20:30Z","extra":true}`)
		requireStatus(t, response, http.StatusBadRequest)
		if set.called {
			t.Fatal("set usecase should not be called")
		}
	})

	t.Run("catalog not found", func(t *testing.T) {
		set := &fakeSetUsecase{err: catalogservice.NotFoundError("coarse unit not found")}
		server := newServer(&fakeStatusUsecase{}, set, &fakeUnsetUsecase{}, &fakeListUsecase{})
		t.Cleanup(server.Close)

		response := putJSON(t, server, "/api/word-favorites", `{"coarse_unit_id":108404,"text":"Making","source":"word_list","occurred_at":"2026-05-24T10:20:30Z"}`)
		requireStatus(t, response, http.StatusNotFound)
	})
}

func TestWordFavoriteBodyLimitMapsPayloadTooLarge(t *testing.T) {
	server := newServerWithBodyLimit(&fakeStatusUsecase{}, &fakeSetUsecase{}, &fakeUnsetUsecase{}, &fakeListUsecase{})
	t.Cleanup(server.Close)

	response := putJSON(t, server, "/api/word-favorites", `{"coarse_unit_id":108404,"text":"`+strings.Repeat("x", 128)+`","source":"word_list","occurred_at":"2026-05-24T10:20:30Z"}`)
	requireStatus(t, response, http.StatusRequestEntityTooLarge)
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response, &body)
	if body.Error.Code != "payload_too_large" {
		t.Fatalf("error code = %q, want payload_too_large", body.Error.Code)
	}
}

func newServer(status *fakeStatusUsecase, set *fakeSetUsecase, unset *fakeUnsetUsecase, list *fakeListUsecase) *httptest.Server {
	handler := router.New(router.Options{
		WordFavorites: wordfavorites.NewHandler(status, set, unset, list),
	})
	handler = auth.PrincipalMiddleware(auth.Options{GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo"})(handler)
	handler = middleware.RequestID(handler)
	return httptest.NewServer(handler)
}

func newServerWithBodyLimit(status *fakeStatusUsecase, set *fakeSetUsecase, unset *fakeUnsetUsecase, list *fakeListUsecase) *httptest.Server {
	handler := router.New(router.Options{
		WordFavorites: wordfavorites.NewHandler(status, set, unset, list),
	})
	handler = middleware.BodyLimitByPath(64, nil)(handler)
	handler = auth.PrincipalMiddleware(auth.Options{GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo"})(handler)
	handler = middleware.RequestID(handler)
	return httptest.NewServer(handler)
}

func postJSON(t *testing.T, server *httptest.Server, path string, body string) *http.Response {
	t.Helper()
	return requestJSON(t, server, http.MethodPost, path, body)
}

func putJSON(t *testing.T, server *httptest.Server, path string, body string) *http.Response {
	t.Helper()
	return requestJSON(t, server, http.MethodPut, path, body)
}

func deleteJSON(t *testing.T, server *httptest.Server, path string, body string) *http.Response {
	t.Helper()
	return requestJSON(t, server, http.MethodDelete, path, body)
}

func requestJSON(t *testing.T, server *httptest.Server, method string, path string, body string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Apigateway-Api-Userinfo", base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"`+userID+`"}`)))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	return response
}

func get(t *testing.T, server *httptest.Server, path string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("X-Apigateway-Api-Userinfo", base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"`+userID+`"}`)))
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
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want %d: %s", response.StatusCode, want, string(body))
	}
}

func requireEmptyBody(t *testing.T, response *http.Response) {
	t.Helper()
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if len(bytes.TrimSpace(body)) != 0 {
		t.Fatalf("body = %q, want empty", string(body))
	}
}

func decodeJSON(t *testing.T, response *http.Response, target any) {
	t.Helper()
	defer response.Body.Close()
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

type fakeStatusUsecase struct {
	called   bool
	request  catalogdto.GetWordFavoriteStatusRequest
	response catalogdto.WordFavoriteStatusResponse
	err      error
}

func (f *fakeStatusUsecase) Execute(ctx context.Context, request catalogdto.GetWordFavoriteStatusRequest) (catalogdto.WordFavoriteStatusResponse, error) {
	f.called = true
	f.request = request
	return f.response, f.err
}

type fakeSetUsecase struct {
	called  bool
	request catalogdto.SetWordFavoriteRequest
	err     error
}

func (f *fakeSetUsecase) Execute(ctx context.Context, request catalogdto.SetWordFavoriteRequest) error {
	f.called = true
	f.request = request
	return f.err
}

type fakeUnsetUsecase struct {
	called  bool
	request catalogdto.UnsetWordFavoriteRequest
	err     error
}

func (f *fakeUnsetUsecase) Execute(ctx context.Context, request catalogdto.UnsetWordFavoriteRequest) error {
	f.called = true
	f.request = request
	return f.err
}

type fakeListUsecase struct {
	called   bool
	request  catalogdto.ListWordFavoritesRequest
	response catalogdto.WordFavoriteListPage
	err      error
}

func (f *fakeListUsecase) Execute(ctx context.Context, request catalogdto.ListWordFavoritesRequest) (catalogdto.WordFavoriteListPage, error) {
	f.called = true
	f.request = request
	return f.response, f.err
}

const (
	userID      = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	wordVideoID = "00000000-0000-4000-8000-000000000001"
)

func int64Ptr(value int64) *int64 {
	return &value
}

func int32Ptr(value int32) *int32 {
	return &value
}

func stringPtr(value string) *string {
	return &value
}
