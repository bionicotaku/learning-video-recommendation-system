package unitcollections_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/unitcollections"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/router"
	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	learningservice "learning-video-recommendation-system/internal/learningengine/reducer/application/service"
	semanticdto "learning-video-recommendation-system/internal/semantic/application/dto"
)

func TestListUnitCollectionsReturnsSemanticResponse(t *testing.T) {
	semantic := &fakeListCollectionsUsecase{response: semanticdto.ListUnitCollectionsResponse{
		Items: []semanticdto.UnitCollectionItem{{
			CollectionID:    "11111111-1111-4111-8111-111111111111",
			Slug:            "toefl-core",
			Name:            "TOEFL Core",
			Category:        "wordbook",
			CoarseUnitCount: 1000,
			WordUnitCount:   1000,
		}},
	}}
	server := newServer(semantic, &fakeActivateUsecase{}, true)
	t.Cleanup(server.Close)

	response := request(t, server, http.MethodGet, "/api/unit-collections", "", false)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	var body map[string]any
	decodeJSON(t, response, &body)
	items := body["items"].([]any)
	item := items[0].(map[string]any)
	if item["slug"] != "toefl-core" || item["coarse_unit_count"].(float64) != 1000 {
		t.Fatalf("unexpected body: %+v", body)
	}
	if !semantic.called {
		t.Fatalf("semantic usecase was not called")
	}
}

func TestActivateUnitCollectionMapsPrincipalAndBody(t *testing.T) {
	activate := &fakeActivateUsecase{response: learningdto.ActivateUnitCollectionTargetResponse{
		CollectionID:   "11111111-1111-4111-8111-111111111111",
		CollectionSlug: "toefl-core",
		TargetCount:    1000,
	}}
	server := newServer(&fakeListCollectionsUsecase{}, activate, true)
	t.Cleanup(server.Close)

	response := request(t, server, http.MethodPut, "/api/learning-targets/active-collection", `{"collection_slug":"toefl-core"}`, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if !activate.called || activate.request.UserID != userID || activate.request.CollectionSlug != "toefl-core" {
		t.Fatalf("request not mapped: %+v", activate.request)
	}
	var body map[string]any
	decodeJSON(t, response, &body)
	if body["collection_slug"] != "toefl-core" || body["target_count"].(float64) != 1000 {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestActivateUnitCollectionRejectsInvalidRequests(t *testing.T) {
	cases := []struct {
		name          string
		body          string
		withPrincipal bool
		wantStatus    int
	}{
		{name: "missing principal", body: `{"collection_slug":"toefl-core"}`, withPrincipal: false, wantStatus: http.StatusUnauthorized},
		{name: "bad json", body: `{`, withPrincipal: true, wantStatus: http.StatusBadRequest},
		{name: "invalid slug", body: `{"collection_slug":"TOEFL Core"}`, withPrincipal: true, wantStatus: http.StatusBadRequest},
		{name: "unknown field", body: `{"collection_slug":"toefl-core","coarse_unit_ids":[1]}`, withPrincipal: true, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			activate := &fakeActivateUsecase{}
			server := newServer(&fakeListCollectionsUsecase{}, activate, true)
			t.Cleanup(server.Close)

			response := request(t, server, http.MethodPut, "/api/learning-targets/active-collection", tt.body, tt.withPrincipal)
			if response.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tt.wantStatus, response.StatusCode, readBody(t, response))
			}
			if activate.called {
				t.Fatalf("activate usecase should not be called")
			}
		})
	}
}

func TestActivateUnitCollectionMapsNotFound(t *testing.T) {
	server := newServer(&fakeListCollectionsUsecase{}, &fakeActivateUsecase{err: learningservice.ErrUnitCollectionNotFound}, true)
	t.Cleanup(server.Close)

	response := request(t, server, http.MethodPut, "/api/learning-targets/active-collection", `{"collection_slug":"missing-book"}`, true)
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.StatusCode, readBody(t, response))
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response, &body)
	if body.Error.Code != "not_found" {
		t.Fatalf("code = %q, want not_found", body.Error.Code)
	}
}

func newServer(list *fakeListCollectionsUsecase, activate *fakeActivateUsecase, withAuth bool) *httptest.Server {
	group := unitcollections.NewHandler(list, activate)
	handler := router.New(router.Options{UnitCollections: group})
	if withAuth {
		handler = auth.PrincipalMiddleware(auth.Options{GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo"})(handler)
	}
	handler = middleware.RequestID(handler)
	return httptest.NewServer(handler)
}

func request(t *testing.T, server *httptest.Server, method string, path string, body string, withPrincipal bool) *http.Response {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	request, err := http.NewRequest(method, server.URL+path, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if withPrincipal {
		request.Header.Set("X-Apigateway-Api-Userinfo", "eyJzdWIiOiIxMTExMTExMS0xMTExLTQxMTEtODExMS0xMTExMTExMTExMTEifQ")
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

type fakeListCollectionsUsecase struct {
	called   bool
	response semanticdto.ListUnitCollectionsResponse
	err      error
}

func (f *fakeListCollectionsUsecase) Execute(context.Context) (semanticdto.ListUnitCollectionsResponse, error) {
	f.called = true
	return f.response, f.err
}

type fakeActivateUsecase struct {
	called   bool
	request  learningdto.ActivateUnitCollectionTargetRequest
	response learningdto.ActivateUnitCollectionTargetResponse
	err      error
}

func (f *fakeActivateUsecase) Execute(_ context.Context, request learningdto.ActivateUnitCollectionTargetRequest) (learningdto.ActivateUnitCollectionTargetResponse, error) {
	f.called = true
	f.request = request
	return f.response, f.err
}

const userID = "11111111-1111-4111-8111-111111111111"

var _ = errors.Is
