package videodetail_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/videodetail"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/router"
	catalogservice "learning-video-recommendation-system/internal/catalog/application/service"
)

func TestVideoDetailReturnsDetailAndPassesPrincipalUserID(t *testing.T) {
	service := &fakeVideoDetailService{
		response: apvdto.VideoDetailResponse{
			VideoID:         "11111111-1111-1111-1111-111111111111",
			Title:           "Title",
			Description:     "Description",
			VideoURL:        "https://cdn.example.com/hls/master.m3u8",
			CoverImageURL:   stringPtr("https://cdn.example.com/covers/111.webp"),
			TranscriptURL:   stringPtr("https://cdn.example.com/transcripts/111.json"),
			DurationSeconds: 91,
			ViewCount:       12,
			LikeCount:       3,
			FavoriteCount:   2,
			UserState:       apvdto.VideoDetailUserState{HasLiked: true, HasFavorited: false},
		},
	}
	server := newServer(service)
	t.Cleanup(server.Close)

	response := getDetail(t, server, "11111111-1111-1111-1111-111111111111", "user-1")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	var body apvdto.VideoDetailResponse
	decodeJSON(t, response, &body)
	if body.VideoID != "11111111-1111-1111-1111-111111111111" || body.VideoURL == "" || body.TranscriptURL == nil {
		t.Fatalf("unexpected body: %+v", body)
	}
	if !body.UserState.HasLiked || body.UserState.HasFavorited {
		t.Fatalf("unexpected user_state: %+v", body.UserState)
	}
	if service.request.UserID != "user-1" || service.request.VideoID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("request not mapped: %+v", service.request)
	}
}

func TestVideoDetailRejectsInvalidVideoIDAndMissingPrincipal(t *testing.T) {
	t.Run("invalid video id", func(t *testing.T) {
		server := newServer(&fakeVideoDetailService{})
		t.Cleanup(server.Close)

		response := getDetail(t, server, "not-a-uuid", "user-1")
		if response.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
		}
	})

	t.Run("missing principal", func(t *testing.T) {
		server := newServer(&fakeVideoDetailService{})
		t.Cleanup(server.Close)

		response := getDetail(t, server, "11111111-1111-1111-1111-111111111111", "")
		if response.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", response.StatusCode, readBody(t, response))
		}
	})
}

func TestVideoDetailMapsErrors(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "invalid", err: apiservice.InvalidRequestError("bad request"), status: http.StatusBadRequest, code: "invalid_request"},
		{name: "not found", err: catalogservice.NotFoundError("video not found"), status: http.StatusNotFound, code: "not_found"},
		{name: "unavailable", err: apiservice.ServiceUnavailableError("timeout"), status: http.StatusServiceUnavailable, code: "service_unavailable"},
		{name: "internal", err: errors.New("db down"), status: http.StatusInternalServerError, code: "internal_error"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			server := newServer(&fakeVideoDetailService{err: tt.err})
			t.Cleanup(server.Close)

			response := getDetail(t, server, "11111111-1111-1111-1111-111111111111", "user-1")
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

func newServer(service *fakeVideoDetailService) *httptest.Server {
	group := videodetail.NewHandler(service)
	handler := router.New(router.Options{VideoDetail: group})
	handler = auth.PrincipalMiddleware(auth.Options{GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo"})(handler)
	handler = middleware.RequestID(handler)
	return httptest.NewServer(handler)
}

func getDetail(t *testing.T, server *httptest.Server, videoID string, userID string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, server.URL+"/api/videos/"+videoID, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if userID == "user-1" {
		request.Header.Set("X-Apigateway-Api-Userinfo", "eyJzdWIiOiJ1c2VyLTEifQ")
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("get: %v", err)
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
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return err.Error()
	}
	encoded, _ := json.Marshal(body)
	return string(encoded)
}

func stringPtr(value string) *string {
	return &value
}

type fakeVideoDetailService struct {
	request  apvdto.GetVideoDetailRequest
	response apvdto.VideoDetailResponse
	err      error
}

func (f *fakeVideoDetailService) Execute(ctx context.Context, request apvdto.GetVideoDetailRequest) (apvdto.VideoDetailResponse, error) {
	f.request = request
	if f.err != nil {
		return apvdto.VideoDetailResponse{}, f.err
	}
	return f.response, nil
}
