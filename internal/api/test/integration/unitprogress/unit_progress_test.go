package unitprogress_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/unitprogress"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/router"
	"learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	learningservice "learning-video-recommendation-system/internal/learningengine/reducer/application/service"
)

func TestUnitProgressMasteredReturnsItemsAndPassesPrincipalUserID(t *testing.T) {
	lastProgressAt := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	pos := "verb"
	chineseLabel := "放弃"
	chineseDef := "停止继续"
	nextCursor := "cursor-1"
	service := &fakeService{response: dto.ListUserUnitProgressResponse{
		Items: []dto.UnitProgressItem{{
			CoarseUnitID:    101,
			Kind:            "word",
			Label:           "abandon",
			Pos:             &pos,
			ChineseLabel:    &chineseLabel,
			ChineseDef:      &chineseDef,
			ProgressPercent: 100,
			LastProgressAt:  &lastProgressAt,
		}},
		Page: dto.UnitProgressPage{Limit: 20, HasMore: true, NextCursor: &nextCursor},
	}}
	server := newServer(service, true)
	t.Cleanup(server.Close)

	response := get(t, server, "/api/learning/unit-progress/mastered?limit=20&cursor=cursor-0", true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if service.request.UserID != "user-1" ||
		service.request.Bucket != dto.UnitProgressBucketMastered ||
		service.request.Limit != 20 ||
		service.request.Cursor != "cursor-0" {
		t.Fatalf("request not mapped: %+v", service.request)
	}

	var body dto.ListUserUnitProgressResponse
	decodeJSON(t, response, &body)
	if len(body.Items) != 1 || body.Items[0].CoarseUnitID != 101 || body.Items[0].Label != "abandon" {
		t.Fatalf("unexpected body: %+v", body)
	}
	if body.Page.NextCursor == nil || *body.Page.NextCursor != "cursor-1" {
		t.Fatalf("page = %+v, want next_cursor", body.Page)
	}
}

func TestUnitProgressUnmasteredReturnsItemsAndPassesPrincipalUserID(t *testing.T) {
	service := &fakeService{response: dto.ListUserUnitProgressResponse{
		Items: []dto.UnitProgressItem{{CoarseUnitID: 201, Kind: "word", Label: "derive", ProgressPercent: 64.25}},
		Page:  dto.UnitProgressPage{Limit: 50},
	}}
	server := newServer(service, true)
	t.Cleanup(server.Close)

	response := get(t, server, "/api/learning/unit-progress/unmastered", true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if service.request.UserID != "user-1" ||
		service.request.Bucket != dto.UnitProgressBucketUnmastered ||
		service.request.Limit != 0 ||
		service.request.Cursor != "" {
		t.Fatalf("request not mapped: %+v", service.request)
	}
}

func TestUnitProgressRejectsMissingPrincipal(t *testing.T) {
	service := &fakeService{}
	server := newServer(service, false)
	t.Cleanup(server.Close)

	response := get(t, server, "/api/learning/unit-progress/mastered", false)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if service.called {
		t.Fatalf("service should not be called")
	}
}

func TestUnitProgressRejectsInvalidLimit(t *testing.T) {
	cases := []string{"0", "101", "abc"}

	for _, limit := range cases {
		t.Run(limit, func(t *testing.T) {
			service := &fakeService{}
			server := newServer(service, true)
			t.Cleanup(server.Close)

			response := get(t, server, "/api/learning/unit-progress/mastered?limit="+limit, true)
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
			}
			if service.called {
				t.Fatalf("service should not be called")
			}
		})
	}
}

func TestUnitProgressMapsServiceErrors(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "validation", err: &learningservice.Error{Code: learningservice.ErrorCodeValidation, Message: "bad cursor"}, status: http.StatusBadRequest, code: "invalid_request"},
		{name: "internal", err: errors.New("db down"), status: http.StatusInternalServerError, code: "internal_error"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			server := newServer(&fakeService{err: tt.err}, true)
			t.Cleanup(server.Close)

			response := get(t, server, "/api/learning/unit-progress/mastered", true)
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
				t.Fatalf("code = %q, want %q", body.Error.Code, tt.code)
			}
		})
	}
}

func newServer(service *fakeService, withAuth bool) *httptest.Server {
	group := unitprogress.NewHandler(service)
	handler := router.New(router.Options{UnitProgress: group})
	if withAuth {
		handler = auth.PrincipalMiddleware(auth.Options{GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo"})(handler)
	}
	handler = middleware.RequestID(handler)
	return httptest.NewServer(handler)
}

func get(t *testing.T, server *httptest.Server, path string, setPrincipal bool) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if setPrincipal {
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
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(response.Body)
	return buf.String()
}

type fakeService struct {
	called   bool
	request  dto.ListUserUnitProgressRequest
	response dto.ListUserUnitProgressResponse
	err      error
}

func (f *fakeService) Execute(_ context.Context, request dto.ListUserUnitProgressRequest) (dto.ListUserUnitProgressResponse, error) {
	f.called = true
	f.request = request
	return f.response, f.err
}
