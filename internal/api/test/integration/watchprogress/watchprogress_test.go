package watchprogress_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/watchprogress"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/router"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
	catalogservice "learning-video-recommendation-system/internal/catalog/application/service"
)

func TestWatchProgressReturnsAcceptedAndPassesPrincipalUserID(t *testing.T) {
	recorder := &fakeRecorder{response: catalogdto.RecordVideoWatchProgressResponse{Accepted: true}}
	server := newServer(recorder)
	t.Cleanup(server.Close)

	response := postJSON(t, server, `{
		"video_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"watch_session_id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		"position_ms": 1000,
		"active_watch_ms": 2000,
		"occurred_at": "2026-05-16T12:00:00Z",
		"source_surface": "fullscreen",
		"client_context": {"platform":"ios"},
		"metadata": {"player":"native"}
	}`)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	var body struct {
		Accepted bool `json:"accepted"`
	}
	decodeJSON(t, response, &body)
	if !body.Accepted {
		t.Fatalf("expected accepted response: %+v", body)
	}
	if recorder.request.UserID != "user-1" {
		t.Fatalf("expected principal user id, got %q", recorder.request.UserID)
	}
	if recorder.request.VideoID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" || recorder.request.WatchSessionID != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Fatalf("unexpected request mapping: %+v", recorder.request)
	}
}

func TestWatchProgressRejectsInvalidTransportRequest(t *testing.T) {
	recorder := &fakeRecorder{}
	server := newServer(recorder)
	t.Cleanup(server.Close)

	response := postJSON(t, server, `{
		"video_id": "not-a-uuid",
		"watch_session_id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		"position_ms": 1000,
		"active_watch_ms": 2000
	}`)

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if recorder.called {
		t.Fatal("recorder should not be called")
	}
}

func TestWatchProgressRequiresContentType(t *testing.T) {
	recorder := &fakeRecorder{}
	server := newServer(recorder)
	t.Cleanup(server.Close)

	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/video-watch-progress", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("X-Apigateway-Api-Userinfo", "eyJzdWIiOiJ1c2VyLTEifQ")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.StatusCode)
	}
}

func TestWatchProgressMapsCatalogErrors(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "not found", err: catalogservice.NotFoundError("video not found"), status: http.StatusNotFound, code: "not_found"},
		{name: "conflict", err: catalogservice.ConflictError("conflict"), status: http.StatusConflict, code: "conflict"},
		{name: "unprocessable", err: catalogservice.UnprocessableError("bad occurred_at"), status: http.StatusUnprocessableEntity, code: "unprocessable_entity"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			server := newServer(&fakeRecorder{err: tt.err})
			t.Cleanup(server.Close)

			response := postJSON(t, server, `{
				"video_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				"watch_session_id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
				"position_ms": 1000,
				"active_watch_ms": 2000
			}`)

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

func TestWatchProgressMapsUnexpectedErrorToInternal(t *testing.T) {
	server := newServer(&fakeRecorder{err: errors.New("db down")})
	t.Cleanup(server.Close)

	response := postJSON(t, server, `{
		"video_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"watch_session_id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		"position_ms": 1000,
		"active_watch_ms": 2000
	}`)
	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", response.StatusCode)
	}
}

func newServer(recorder *fakeRecorder) *httptest.Server {
	group := watchprogress.NewHandler(recorder)
	handler := router.New(router.Options{WatchProgress: group})
	handler = auth.PrincipalMiddleware(auth.Options{GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo"})(handler)
	handler = middleware.RequestID(handler)
	return httptest.NewServer(handler)
}

func postJSON(t *testing.T, server *httptest.Server, body string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/video-watch-progress", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Apigateway-Api-Userinfo", "eyJzdWIiOiJ1c2VyLTEifQ")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post: %v", err)
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

type fakeRecorder struct {
	called   bool
	request  catalogdto.RecordVideoWatchProgressRequest
	response catalogdto.RecordVideoWatchProgressResponse
	err      error
}

func (f *fakeRecorder) Execute(ctx context.Context, request catalogdto.RecordVideoWatchProgressRequest) (catalogdto.RecordVideoWatchProgressResponse, error) {
	f.called = true
	f.request = request
	return f.response, f.err
}
