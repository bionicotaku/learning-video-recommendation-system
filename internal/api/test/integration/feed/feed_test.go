package feed_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/feed"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/router"
)

func TestFeedReturnsItemsAndPassesPrincipalUserID(t *testing.T) {
	service := &fakeFeedService{
		response: apvdto.FeedResponse{
			RecommendationRunID: "run-1",
			Items: []apvdto.FeedItem{
				{
					VideoID:         "11111111-1111-1111-1111-111111111111",
					Title:           "Title",
					CoverImageURL:   stringPtr("https://cdn.example.com/covers/111.webp"),
					DurationSeconds: 91,
					ViewCount:       12,
					LearningUnits: []apvdto.FeedLearningUnit{
						{CoarseUnitID: 101, Text: "serendipity", Role: "hard_review", IsPrimary: true, EvidenceSentenceIndex: 2, EvidenceSpanIndex: 1, EvidenceStartMS: 2000, EvidenceEndMS: 2400},
					},
				},
			},
		},
	}
	server := newServer(service)
	t.Cleanup(server.Close)

	response := postJSON(t, server, `{
		"target_video_count": 6,
		"client_context": {"platform":"ios"}
	}`)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	var body apvdto.FeedResponse
	decodeJSON(t, response, &body)
	if body.RecommendationRunID != "run-1" || len(body.Items) != 1 {
		t.Fatalf("unexpected response: %+v", body)
	}
	if service.request.UserID != "user-1" || service.request.TargetVideoCount != 6 {
		t.Fatalf("request not mapped: %+v", service.request)
	}
}

func TestFeedResponseDoesNotExposeVideoDetailFields(t *testing.T) {
	service := &fakeFeedService{
		response: apvdto.FeedResponse{
			RecommendationRunID: "run-1",
			Items: []apvdto.FeedItem{
				{
					VideoID:         "11111111-1111-1111-1111-111111111111",
					Title:           "Title",
					DurationSeconds: 91,
					ViewCount:       12,
					LearningUnits:   []apvdto.FeedLearningUnit{},
				},
			},
		},
	}
	server := newServer(service)
	t.Cleanup(server.Close)

	response := postJSON(t, server, `{}`)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}

	var raw struct {
		Items []map[string]any `json:"items"`
	}
	decodeJSON(t, response, &raw)
	if len(raw.Items) != 1 {
		t.Fatalf("items = %+v, want one item", raw.Items)
	}
	for _, field := range []string{"description", "video_url", "transcript_url", "like_count", "favorite_count", "has_liked", "has_favorited"} {
		if _, ok := raw.Items[0][field]; ok {
			t.Fatalf("feed item exposed removed field %q: %+v", field, raw.Items[0])
		}
	}
}

func TestFeedAppliesRequestDefaults(t *testing.T) {
	service := &fakeFeedService{response: apvdto.FeedResponse{RecommendationRunID: "run-1"}}
	server := newServer(service)
	t.Cleanup(server.Close)

	response := postJSON(t, server, `{}`)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if service.request.TargetVideoCount != 8 {
		t.Fatalf("defaults not applied: %+v", service.request)
	}
	if string(service.request.ClientContext) != "{}" {
		t.Fatalf("default client_context = %s", service.request.ClientContext)
	}
}

func TestFeedRejectsRemovedRequestFields(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "preferred duration", body: `{"preferred_duration_sec":{"min":15,"max":90}}`},
		{name: "session hint", body: `{"session_hint":"mixed"}`},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeFeedService{}
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

func TestFeedRejectsInvalidTransportRequest(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "zero target count", body: `{"target_video_count": 0}`},
		{name: "too many target videos", body: `{"target_video_count": 21}`},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeFeedService{}
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

func TestFeedMapsOversizeJSONBodyToPayloadTooLarge(t *testing.T) {
	service := &fakeFeedService{}
	server := newServerWithBodyLimit(service)
	t.Cleanup(server.Close)

	body := append([]byte(`{"client_context":{"padding":"`), bytes.Repeat([]byte("x"), 1<<20)...)
	body = append(body, []byte(`"}}`)...)
	response := postJSONBytes(t, server, body)

	if response.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d: %s", response.StatusCode, readBody(t, response))
	}
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response, &payload)
	if payload.Error.Code != "payload_too_large" {
		t.Fatalf("code = %q, want payload_too_large", payload.Error.Code)
	}
	if service.called {
		t.Fatal("service should not be called")
	}
}

func TestFeedRequiresContentType(t *testing.T) {
	service := &fakeFeedService{}
	server := newServer(service)
	t.Cleanup(server.Close)

	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/feed", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	setGatewayPrincipal(request, "user-1")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.StatusCode)
	}
}

func TestFeedMapsErrors(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "invalid", err: apiservice.InvalidRequestError("bad request"), status: http.StatusBadRequest, code: "invalid_request"},
		{name: "unavailable", err: apiservice.ServiceUnavailableError("timeout"), status: http.StatusServiceUnavailable, code: "service_unavailable"},
		{name: "internal", err: errors.New("db down"), status: http.StatusInternalServerError, code: "internal_error"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			server := newServer(&fakeFeedService{err: tt.err})
			t.Cleanup(server.Close)

			response := postJSON(t, server, `{}`)
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

func newServer(feedService *fakeFeedService) *httptest.Server {
	return newServerWithAuth(feedService, auth.Options{GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo"})
}

func newServerWithAuth(feedService *fakeFeedService, options auth.Options) *httptest.Server {
	group := feed.NewHandler(feedService)
	handler := router.New(router.Options{Feed: group})
	handler = auth.PrincipalMiddleware(options)(handler)
	handler = middleware.RequestID(handler)
	return httptest.NewServer(handler)
}

func newServerWithBodyLimit(feedService *fakeFeedService) *httptest.Server {
	group := feed.NewHandler(feedService)
	handler := router.New(router.Options{Feed: group})
	handler = middleware.BodyLimitByPath(1<<20, nil)(handler)
	handler = auth.PrincipalMiddleware(auth.Options{GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo"})(handler)
	handler = middleware.RequestID(handler)
	return httptest.NewServer(handler)
}

func postJSON(t *testing.T, server *httptest.Server, body string) *http.Response {
	t.Helper()
	return postJSONBytes(t, server, []byte(body))
}

func postJSONBytes(t *testing.T, server *httptest.Server, body []byte) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/feed", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	setGatewayPrincipal(request, "user-1")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	return response
}

func setGatewayPrincipal(request *http.Request, userID string) {
	switch userID {
	case "user-1":
		request.Header.Set("X-Apigateway-Api-Userinfo", "eyJzdWIiOiJ1c2VyLTEifQ")
	default:
		panic("unsupported test user id")
	}
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

func stringPtr(value string) *string {
	return &value
}

type fakeFeedService struct {
	called   bool
	request  apvdto.GetFeedRequest
	response apvdto.FeedResponse
	err      error
}

func (f *fakeFeedService) Execute(ctx context.Context, request apvdto.GetFeedRequest) (apvdto.FeedResponse, error) {
	f.called = true
	f.request = request
	return f.response, f.err
}
