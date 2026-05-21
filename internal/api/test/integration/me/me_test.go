package me_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	mehandler "learning-video-recommendation-system/internal/api/infrastructure/http/handler/me"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/router"
	userdto "learning-video-recommendation-system/internal/user/application/dto"
)

func TestMeReturnsProfileStatsAndUpdatesTimezoneHeader(t *testing.T) {
	displayName := "alice"
	timezone := "Asia/Shanghai"
	service := &fakeMeService{response: userdto.MeResponse{
		UserID:           "user-1",
		Email:            stringPtr("alice@example.com"),
		EmailConfirmed:   true,
		DisplayName:      &displayName,
		Locale:           "zh-CN",
		Timezone:         &timezone,
		OnboardingStatus: "new",
		Stats: userdto.MeStats{
			TotalWatchSeconds: 3600,
			QuizAttemptCount:  12,
			StartedUnitCount:  48,
		},
	}}
	server := newServer(service, &fakeCalendarService{}, true)
	t.Cleanup(server.Close)

	response := get(t, server, "/api/me", true, "Asia/Shanghai")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if service.request.UserID != "user-1" || service.request.ClientTimezone != "Asia/Shanghai" {
		t.Fatalf("request not mapped: %+v", service.request)
	}

	var body userdto.MeResponse
	decodeJSON(t, response, &body)
	if body.UserID != "user-1" || body.Email == nil || *body.Email != "alice@example.com" || body.Stats.StartedUnitCount != 48 {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestActivityCalendarReturnsSevenDaysAndUsesTimezoneHeader(t *testing.T) {
	service := &fakeCalendarService{response: userdto.ActivityCalendarResponse{
		Timezone: "Asia/Shanghai",
		Today:    "2026-05-21",
		Days: []userdto.ActivityDay{
			{LocalDate: "2026-05-15"},
			{LocalDate: "2026-05-16", WatchSeconds: 30, IsActive: true},
			{LocalDate: "2026-05-17"},
			{LocalDate: "2026-05-18"},
			{LocalDate: "2026-05-19"},
			{LocalDate: "2026-05-20"},
			{LocalDate: "2026-05-21", QuizAttemptCount: 1, LearningInteractionCount: 2, IsActive: true},
		},
	}}
	server := newServer(&fakeMeService{}, service, true)
	t.Cleanup(server.Close)

	response := get(t, server, "/api/me/activity-calendar", true, "Asia/Shanghai")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if service.request.UserID != "user-1" || service.request.ClientTimezone != "Asia/Shanghai" {
		t.Fatalf("request not mapped: %+v", service.request)
	}

	var body userdto.ActivityCalendarResponse
	decodeJSON(t, response, &body)
	if body.Timezone != "Asia/Shanghai" || len(body.Days) != 7 || body.Days[1].WatchSeconds != 30 || !body.Days[6].IsActive {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestMeRequiresPrincipal(t *testing.T) {
	service := &fakeMeService{}
	server := newServer(service, &fakeCalendarService{}, true)
	t.Cleanup(server.Close)

	response := get(t, server, "/api/me", false, "")
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if service.called {
		t.Fatalf("service should not be called")
	}
}

func newServer(meService *fakeMeService, calendarService *fakeCalendarService, withAuth bool) *httptest.Server {
	group := mehandler.NewHandler(meService, calendarService)
	handler := router.New(router.Options{Me: group})
	if withAuth {
		handler = auth.PrincipalMiddleware(auth.Options{GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo"})(handler)
	}
	handler = middleware.RequestID(handler)
	return httptest.NewServer(handler)
}

func get(t *testing.T, server *httptest.Server, path string, setPrincipal bool, timezone string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if setPrincipal {
		request.Header.Set("X-Apigateway-Api-Userinfo", "eyJzdWIiOiJ1c2VyLTEifQ")
	}
	if timezone != "" {
		request.Header.Set("X-Client-Timezone", timezone)
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

func stringPtr(value string) *string {
	return &value
}

type fakeMeService struct {
	called   bool
	request  userdto.MeRequest
	response userdto.MeResponse
}

func (f *fakeMeService) Execute(_ context.Context, request userdto.MeRequest) (userdto.MeResponse, error) {
	f.called = true
	f.request = request
	return f.response, nil
}

type fakeCalendarService struct {
	called   bool
	request  userdto.ActivityCalendarRequest
	response userdto.ActivityCalendarResponse
}

func (f *fakeCalendarService) Execute(_ context.Context, request userdto.ActivityCalendarRequest) (userdto.ActivityCalendarResponse, error) {
	f.called = true
	f.request = request
	return f.response, nil
}
