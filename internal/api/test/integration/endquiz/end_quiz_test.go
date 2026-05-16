package endquiz_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/endquiz"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/router"
	catalogdto "learning-video-recommendation-system/internal/catalog/application/dto"
	catalogservice "learning-video-recommendation-system/internal/catalog/application/service"
)

func TestEndQuizReturnsItemsAndMapsRequest(t *testing.T) {
	contextSentenceIndex := int32(3)
	service := &fakeEndQuizService{
		response: catalogdto.EndQuizQuestionLookupResponse{
			VideoID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			Items: []catalogdto.EndQuizItem{
				{
					CoarseUnitID:         101,
					QuestionID:           "11111111-1111-1111-1111-111111111111",
					Source:               "video_context",
					QuestionType:         "context_meaning_choice",
					TargetText:           "serendipity",
					Question:             "What does it mean?",
					Options:              []catalogdto.EndQuizOption{{OptionID: "correct", Text: "right"}},
					ContextSentenceIndex: &contextSentenceIndex,
				},
			},
			MissingCoarseUnitIDs: []int64{999},
		},
	}
	server := newServer(service)
	t.Cleanup(server.Close)

	response := postJSON(t, server, `{
		"video_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"coarse_unit_ids": [101, 102, 101],
		"recommendation_run_id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		"client_context": {"surface":"fullscreen"}
	}`)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	var body catalogdto.EndQuizQuestionLookupResponse
	decodeJSON(t, response, &body)
	if body.VideoID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" || len(body.Items) != 1 || len(body.MissingCoarseUnitIDs) != 1 {
		t.Fatalf("unexpected response: %+v", body)
	}
	if service.request.VideoID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("video_id not mapped: %+v", service.request)
	}
	if len(service.request.CoarseUnitIDs) != 2 || service.request.CoarseUnitIDs[0] != 101 || service.request.CoarseUnitIDs[1] != 102 {
		t.Fatalf("coarse_unit_ids should be deduped preserving order: %+v", service.request.CoarseUnitIDs)
	}
}

func TestEndQuizRejectsInvalidTransportRequest(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "missing video", body: `{"coarse_unit_ids":[101]}`},
		{name: "invalid video", body: `{"video_id":"bad","coarse_unit_ids":[101]}`},
		{name: "empty units", body: `{"video_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","coarse_unit_ids":[]}`},
		{name: "too many units", body: `{"video_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","coarse_unit_ids":[1,2,3,4,5,6,7,8,9]}`},
		{name: "non-positive unit", body: `{"video_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","coarse_unit_ids":[0]}`},
		{name: "invalid recommendation run", body: `{"video_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","coarse_unit_ids":[101],"recommendation_run_id":"bad"}`},
		{name: "invalid client context", body: `{"video_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","coarse_unit_ids":[101],"client_context":[]}`},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeEndQuizService{}
			server := newServer(service)
			t.Cleanup(server.Close)

			response := postJSON(t, server, tt.body)
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
			}
			if service.called {
				t.Fatal("service should not be called")
			}
		})
	}
}

func TestEndQuizRequiresPrincipal(t *testing.T) {
	service := &fakeEndQuizService{}
	server := newServer(service)
	t.Cleanup(server.Close)

	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/videos/end-quiz", bytes.NewBufferString(`{"video_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","coarse_unit_ids":[101]}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", response.StatusCode, readBody(t, response))
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response, &body)
	if body.Error.Code != "unauthorized" {
		t.Fatalf("expected unauthorized, got %q", body.Error.Code)
	}
	if service.called {
		t.Fatal("service should not be called")
	}
}

func TestEndQuizMapsErrors(t *testing.T) {
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
			server := newServer(&fakeEndQuizService{err: tt.err})
			t.Cleanup(server.Close)

			response := postJSON(t, server, `{"video_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","coarse_unit_ids":[101]}`)
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

func newServer(service *fakeEndQuizService) *httptest.Server {
	group := endquiz.NewHandler(service)
	handler := router.New(router.Options{EndQuiz: group})
	handler = auth.TrustedHeaderPrincipalMiddleware("X-Trusted-User-ID")(handler)
	handler = middleware.RequestID(handler)
	return httptest.NewServer(handler)
}

func postJSON(t *testing.T, server *httptest.Server, body string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/videos/end-quiz", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Trusted-User-ID", "user-1")
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

type fakeEndQuizService struct {
	called   bool
	request  catalogdto.EndQuizQuestionLookupRequest
	response catalogdto.EndQuizQuestionLookupResponse
	err      error
}

func (f *fakeEndQuizService) Execute(ctx context.Context, request catalogdto.EndQuizQuestionLookupRequest) (catalogdto.EndQuizQuestionLookupResponse, error) {
	f.called = true
	f.request = request
	return f.response, f.err
}
