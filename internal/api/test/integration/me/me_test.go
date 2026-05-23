package me_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	mehandler "learning-video-recommendation-system/internal/api/infrastructure/http/handler/me"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/router"
	userdto "learning-video-recommendation-system/internal/user/application/dto"
	userservice "learning-video-recommendation-system/internal/user/application/service"
)

func TestMeReturnsProfileStatsAndUpdatesTimezoneHeader(t *testing.T) {
	timezone := "Asia/Shanghai"
	service := &fakeMeService{response: userdto.MeResponse{
		UserID:           "user-1",
		Email:            stringPtr("alice@example.com"),
		EmailConfirmed:   true,
		DisplayName:      "alice",
		Locale:           "zh-CN",
		Timezone:         &timezone,
		OnboardingStatus: "new",
		BirthDate:        stringPtr("1998-03-14"),
		Gender:           stringPtr("prefer_not_to_say"),
		EducationStage:   stringPtr("undergraduate"),
		IPRegion:         stringPtr("CN-GD"),
		Stats: userdto.MeStats{
			TotalWatchSeconds: 3600,
			QuizAttemptCount:  12,
			StartedUnitCount:  48,
		},
		ActivityCalendar: userdto.ActivityCalendar{
			Timezone:          "Asia/Shanghai",
			Today:             "2026-05-21",
			CurrentStreakDays: 3,
			Days: []userdto.ActivityDay{
				{LocalDate: "2026-05-15"},
				{LocalDate: "2026-05-16", WatchSeconds: 30},
				{LocalDate: "2026-05-17"},
				{LocalDate: "2026-05-18"},
				{LocalDate: "2026-05-19"},
				{LocalDate: "2026-05-20"},
				{LocalDate: "2026-05-21", QuizAttemptCount: 1, LearningInteractionCount: 2},
			},
		},
	}}
	server := newServer(service, true)
	t.Cleanup(server.Close)

	response := get(t, server, "/api/me", true, "Asia/Shanghai")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if service.request.UserID != "user-1" || service.request.ClientTimezone != "Asia/Shanghai" {
		t.Fatalf("request not mapped: %+v", service.request)
	}

	payload := readBody(t, response)
	if bytes.Contains([]byte(payload), []byte("is_active")) {
		t.Fatalf("response should not include is_active: %s", payload)
	}

	var body userdto.MeResponse
	if err := json.Unmarshal([]byte(payload), &body); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if body.UserID != "user-1" || body.Email == nil || *body.Email != "alice@example.com" || body.Stats.StartedUnitCount != 48 {
		t.Fatalf("unexpected body: %+v", body)
	}
	if body.DisplayName != "alice" ||
		body.BirthDate == nil || *body.BirthDate != "1998-03-14" ||
		body.Gender == nil || *body.Gender != "prefer_not_to_say" ||
		body.EducationStage == nil || *body.EducationStage != "undergraduate" ||
		body.IPRegion == nil || *body.IPRegion != "CN-GD" {
		t.Fatalf("unexpected profile fields: %+v", body)
	}
	if body.ActivityCalendar.Timezone != "Asia/Shanghai" ||
		body.ActivityCalendar.CurrentStreakDays != 3 ||
		len(body.ActivityCalendar.Days) != 7 ||
		body.ActivityCalendar.Days[1].WatchSeconds != 30 {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestMeRequiresPrincipal(t *testing.T) {
	service := &fakeMeService{}
	server := newServer(service, true)
	t.Cleanup(server.Close)

	response := get(t, server, "/api/me", false, "")
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if service.called {
		t.Fatalf("service should not be called")
	}
}

func TestActivityCalendarRouteIsNotRegistered(t *testing.T) {
	server := newServer(&fakeMeService{}, true)
	t.Cleanup(server.Close)

	response := get(t, server, "/api/me/activity-calendar", true, "Asia/Shanghai")
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.StatusCode, readBody(t, response))
	}
}

func TestPatchMeProfileRequiresPrincipal(t *testing.T) {
	updateService := &fakeUpdateProfileService{}
	server := newServerWithUpdate(&fakeMeService{}, updateService, true)
	t.Cleanup(server.Close)

	response := patch(t, server, "/api/me/profile", false, "application/json", `{"display_name":"alice"}`)

	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if updateService.called {
		t.Fatalf("service should not be called")
	}
}

func TestPatchMeProfileMapsRequestAndReturnsProfile(t *testing.T) {
	updateService := &fakeUpdateProfileService{response: userdto.UpdateMeProfileResponse{
		UserID:           "user-1",
		Email:            stringPtr("alice@example.com"),
		EmailConfirmed:   true,
		DisplayName:      "Alice_01",
		Locale:           "zh-CN",
		Timezone:         stringPtr("Asia/Shanghai"),
		OnboardingStatus: "collection_selected",
		BirthDate:        stringPtr("2001-09-01"),
		Gender:           stringPtr("prefer_not_to_say"),
		EducationStage:   stringPtr("primary_school"),
	}}
	server := newServerWithUpdate(&fakeMeService{}, updateService, true)
	t.Cleanup(server.Close)

	response := patch(t, server, "/api/me/profile", true, "application/json", `{
		"display_name": " Alice_01 ",
		"birth_date": "2001-09-01",
		"gender": "prefer_not_to_say",
		"education_stage": "primary_school",
		"timezone": "Asia/Shanghai"
	}`)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if !updateService.called {
		t.Fatalf("service should be called")
	}
	request := updateService.request
	if request.UserID != "user-1" ||
		!request.SetDisplayName || request.DisplayName != " Alice_01 " ||
		!request.SetBirthDate || request.BirthDate == nil || *request.BirthDate != "2001-09-01" ||
		!request.SetGender || request.Gender == nil || *request.Gender != "prefer_not_to_say" ||
		!request.SetEducationStage || request.EducationStage == nil || *request.EducationStage != "primary_school" ||
		!request.SetTimezone || request.Timezone == nil || *request.Timezone != "Asia/Shanghai" {
		t.Fatalf("request not mapped: %+v", request)
	}

	payload := readBody(t, response)
	if bytes.Contains([]byte(payload), []byte("activity_calendar")) || bytes.Contains([]byte(payload), []byte("stats")) {
		t.Fatalf("patch response should not include stats/calendar: %s", payload)
	}
	var body userdto.UpdateMeProfileResponse
	if err := json.Unmarshal([]byte(payload), &body); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if body.DisplayName != "Alice_01" || body.BirthDate == nil || *body.BirthDate != "2001-09-01" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestPatchMeProfileMapsExplicitNulls(t *testing.T) {
	updateService := &fakeUpdateProfileService{response: userdto.UpdateMeProfileResponse{
		UserID:           "user-1",
		DisplayName:      "alice",
		Locale:           "zh-CN",
		OnboardingStatus: "new",
	}}
	server := newServerWithUpdate(&fakeMeService{}, updateService, true)
	t.Cleanup(server.Close)

	response := patch(t, server, "/api/me/profile", true, "application/json", `{
		"birth_date": null,
		"gender": null,
		"education_stage": null,
		"timezone": null
	}`)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	request := updateService.request
	if !request.SetBirthDate || request.BirthDate != nil ||
		!request.SetGender || request.Gender != nil ||
		!request.SetEducationStage || request.EducationStage != nil ||
		!request.SetTimezone || request.Timezone != nil {
		t.Fatalf("null fields not mapped: %+v", request)
	}
}

func TestPatchMeProfileRejectsInvalidTransportRequests(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "missing_content_type", body: `{"display_name":"alice"}`},
		{name: "wrong_content_type", contentType: "text/plain", body: `{"display_name":"alice"}`},
		{name: "empty_object", contentType: "application/json", body: `{}`},
		{name: "invalid_json", contentType: "application/json", body: `{"display_name":`},
		{name: "non_object", contentType: "application/json", body: `[]`},
		{name: "display_name_null", contentType: "application/json", body: `{"display_name":null}`},
		{name: "unknown_field", contentType: "application/json", body: `{"display_name":"alice","email":"bad@example.com"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateService := &fakeUpdateProfileService{}
			server := newServerWithUpdate(&fakeMeService{}, updateService, true)
			t.Cleanup(server.Close)

			response := patch(t, server, "/api/me/profile", true, tt.contentType, tt.body)

			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
			}
			if updateService.called {
				t.Fatalf("service should not be called")
			}
		})
	}
}

func TestPatchMeProfileMapsValidationError(t *testing.T) {
	updateService := &fakeUpdateProfileService{err: userservice.ValidationError("display_name is invalid")}
	server := newServerWithUpdate(&fakeMeService{}, updateService, true)
	t.Cleanup(server.Close)

	response := patch(t, server, "/api/me/profile", true, "application/json", `{"display_name":"alice"}`)

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
	}
}

func newServer(meService *fakeMeService, withAuth bool) *httptest.Server {
	return newServerWithUpdate(meService, &fakeUpdateProfileService{}, withAuth)
}

func newServerWithUpdate(meService *fakeMeService, updateService *fakeUpdateProfileService, withAuth bool) *httptest.Server {
	group := mehandler.NewHandler(meService, updateService)
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

func patch(t *testing.T, server *httptest.Server, path string, setPrincipal bool, contentType string, body string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPatch, server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if setPrincipal {
		request.Header.Set("X-Apigateway-Api-Userinfo", "eyJzdWIiOiJ1c2VyLTEifQ")
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	return response
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

type fakeUpdateProfileService struct {
	called   bool
	request  userdto.UpdateMeProfileRequest
	response userdto.UpdateMeProfileResponse
	err      error
}

func (f *fakeUpdateProfileService) Execute(_ context.Context, request userdto.UpdateMeProfileRequest) (userdto.UpdateMeProfileResponse, error) {
	f.called = true
	f.request = request
	if f.err != nil {
		return userdto.UpdateMeProfileResponse{}, f.err
	}
	return f.response, nil
}
