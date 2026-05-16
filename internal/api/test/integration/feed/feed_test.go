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
					Description:     "Description",
					VideoURL:        "https://cdn.example.com/hls/master.m3u8",
					DurationSeconds: 91,
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
		"preferred_duration_sec": {"min": 15, "max": 90},
		"session_hint": "mixed",
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
	if service.request.UserID != "user-1" || service.request.TargetVideoCount != 6 || service.request.PreferredDurationSec != [2]int{15, 90} {
		t.Fatalf("request not mapped: %+v", service.request)
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
	if service.request.TargetVideoCount != 8 || service.request.PreferredDurationSec != [2]int{45, 180} {
		t.Fatalf("defaults not applied: %+v", service.request)
	}
	if string(service.request.ClientContext) != "{}" {
		t.Fatalf("default client_context = %s", service.request.ClientContext)
	}
}

func TestFeedAppliesPartialPreferredDurationDefaults(t *testing.T) {
	cases := []struct {
		name string
		body string
		want [2]int
	}{
		{name: "min only", body: `{"preferred_duration_sec":{"min":15}}`, want: [2]int{15, 180}},
		{name: "max only", body: `{"preferred_duration_sec":{"max":90}}`, want: [2]int{45, 90}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeFeedService{response: apvdto.FeedResponse{RecommendationRunID: "run-1"}}
			server := newServer(service)
			t.Cleanup(server.Close)

			response := postJSON(t, server, tt.body)
			if response.StatusCode != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
			}
			if service.request.PreferredDurationSec != tt.want {
				t.Fatalf("preferred_duration_sec = %+v, want %+v", service.request.PreferredDurationSec, tt.want)
			}
		})
	}
}

func TestFeedRejectsInvalidTransportRequest(t *testing.T) {
	service := &fakeFeedService{}
	server := newServer(service)
	t.Cleanup(server.Close)

	response := postJSON(t, server, `{"target_video_count": 21}`)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if service.called {
		t.Fatal("service should not be called")
	}
}

func TestFeedRejectsInvalidPreferredDuration(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "negative min", body: `{"preferred_duration_sec":{"min":-1,"max":90}}`},
		{name: "negative max", body: `{"preferred_duration_sec":{"min":15,"max":-1}}`},
		{name: "zero min", body: `{"preferred_duration_sec":{"min":0,"max":90}}`},
		{name: "zero max", body: `{"preferred_duration_sec":{"min":15,"max":0}}`},
		{name: "max less than min", body: `{"preferred_duration_sec":{"min":90,"max":15}}`},
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

func TestFeedRequiresContentType(t *testing.T) {
	service := &fakeFeedService{}
	server := newServer(service)
	t.Cleanup(server.Close)

	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/feed", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("X-Trusted-User-ID", "user-1")
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
	group := feed.NewHandler(feedService)
	handler := router.New(router.Options{Feed: group})
	handler = auth.TrustedHeaderPrincipalMiddleware("X-Trusted-User-ID")(handler)
	handler = middleware.RequestID(handler)
	return httptest.NewServer(handler)
}

func postJSON(t *testing.T, server *httptest.Server, body string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/feed", bytes.NewBufferString(body))
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
