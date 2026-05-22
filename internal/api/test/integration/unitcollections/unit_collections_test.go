package unitcollections_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apivdto "learning-video-recommendation-system/internal/api/application/dto"
	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/learningtargets"
	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/unitcollections"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/router"
	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	learningservice "learning-video-recommendation-system/internal/learningengine/reducer/application/service"
)

func TestListUnitCollectionsReturnsItemsAndActiveCollectionSlug(t *testing.T) {
	activeCollection := "toefl-core"
	list := &fakeListCollectionsForUserUsecase{response: apivdto.UnitCollectionsResponse{
		ActiveCollection: &activeCollection,
		Items: []apivdto.UnitCollectionItem{{
			CollectionID:    "11111111-1111-4111-8111-111111111111",
			Slug:            "toefl-core",
			Name:            "TOEFL Core",
			Category:        "wordbook",
			CoarseUnitCount: 1000,
			WordUnitCount:   1000,
		}},
	}}
	server := newServer(list, &fakeActivateUsecase{}, &fakeActiveTargetsUsecase{}, true)
	t.Cleanup(server.Close)

	response := request(t, server, http.MethodGet, "/api/unit-collections", "", true)
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
	if body["active_collection"] != "toefl-core" {
		t.Fatalf("active_collection = %v, want toefl-core", body["active_collection"])
	}
	if !list.called || list.request.UserID != userID {
		t.Fatalf("list request not mapped: called=%v request=%+v", list.called, list.request)
	}
}

func TestListUnitCollectionsRequiresPrincipal(t *testing.T) {
	list := &fakeListCollectionsForUserUsecase{}
	server := newServer(list, &fakeActivateUsecase{}, &fakeActiveTargetsUsecase{}, true)
	t.Cleanup(server.Close)

	response := request(t, server, http.MethodGet, "/api/unit-collections", "", false)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if list.called {
		t.Fatalf("list usecase should not be called without principal")
	}
}

func TestListUnitCollectionsAllowsNullActiveCollection(t *testing.T) {
	list := &fakeListCollectionsForUserUsecase{response: apivdto.UnitCollectionsResponse{
		Items: []apivdto.UnitCollectionItem{{
			CollectionID: "11111111-1111-4111-8111-111111111111",
			Slug:         "toefl-core",
			Name:         "TOEFL Core",
			Category:     "wordbook",
		}},
	}}
	server := newServer(list, &fakeActivateUsecase{}, &fakeActiveTargetsUsecase{}, true)
	t.Cleanup(server.Close)

	response := request(t, server, http.MethodGet, "/api/unit-collections", "", true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	var body map[string]any
	decodeJSON(t, response, &body)
	if _, ok := body["active_collection"]; !ok {
		t.Fatalf("active_collection field missing: %+v", body)
	}
	if body["active_collection"] != nil {
		t.Fatalf("active_collection = %v, want null", body["active_collection"])
	}
}

func TestActivateUnitCollectionMapsPrincipalAndBody(t *testing.T) {
	activate := &fakeActivateUsecase{response: learningdto.ActivateUnitCollectionTargetResponse{
		CollectionID:   "11111111-1111-4111-8111-111111111111",
		CollectionSlug: "toefl-core",
		TargetCount:    1000,
	}}
	server := newServer(&fakeListCollectionsForUserUsecase{}, activate, &fakeActiveTargetsUsecase{}, true)
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
			server := newServer(&fakeListCollectionsForUserUsecase{}, activate, &fakeActiveTargetsUsecase{}, true)
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

func TestActivateUnitCollectionRequiresJSONContentType(t *testing.T) {
	cases := []struct {
		name        string
		contentType string
	}{
		{name: "missing content type"},
		{name: "text plain", contentType: "text/plain"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			activate := &fakeActivateUsecase{}
			server := newServer(&fakeListCollectionsForUserUsecase{}, activate, &fakeActiveTargetsUsecase{}, true)
			t.Cleanup(server.Close)

			request, err := http.NewRequest(http.MethodPut, server.URL+"/api/learning-targets/active-collection", strings.NewReader(`{"collection_slug":"toefl-core"}`))
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			if tt.contentType != "" {
				request.Header.Set("Content-Type", tt.contentType)
			}
			request.Header.Set("X-Apigateway-Api-Userinfo", "eyJzdWIiOiIxMTExMTExMS0xMTExLTQxMTEtODExMS0xMTExMTExMTExMTEifQ")

			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
			}
			if activate.called {
				t.Fatalf("activate usecase should not be called")
			}
		})
	}
}

func TestActivateUnitCollectionMapsNotFound(t *testing.T) {
	server := newServer(&fakeListCollectionsForUserUsecase{}, &fakeActivateUsecase{err: learningservice.ErrUnitCollectionNotFound}, &fakeActiveTargetsUsecase{}, true)
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

func TestActiveCoarseUnitIDsMapsPrincipalAndResponse(t *testing.T) {
	activeCollection := "toefl-core"
	activeTargets := &fakeActiveTargetsUsecase{response: learningdto.GetActiveLearningTargetCoarseUnitIDsResponse{
		ActiveCollection: &activeCollection,
		TargetCount:      3,
		CoarseUnitIDs:    []int64{101, 205, 309},
	}}
	server := newServer(&fakeListCollectionsForUserUsecase{}, &fakeActivateUsecase{}, activeTargets, true)
	t.Cleanup(server.Close)

	response := request(t, server, http.MethodGet, "/api/learning-targets/active-coarse-unit-ids", "", true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if !activeTargets.called || activeTargets.request.UserID != userID {
		t.Fatalf("request not mapped: %+v", activeTargets.request)
	}
	var body map[string]any
	decodeJSON(t, response, &body)
	if body["active_collection"] != "toefl-core" || body["target_count"].(float64) != 3 {
		t.Fatalf("unexpected body: %+v", body)
	}
	ids := body["coarse_unit_ids"].([]any)
	if len(ids) != 3 || ids[0].(float64) != 101 || ids[1].(float64) != 205 || ids[2].(float64) != 309 {
		t.Fatalf("coarse_unit_ids = %+v", ids)
	}
}

func TestActiveCoarseUnitIDsRequiresPrincipal(t *testing.T) {
	activeTargets := &fakeActiveTargetsUsecase{}
	server := newServer(&fakeListCollectionsForUserUsecase{}, &fakeActivateUsecase{}, activeTargets, true)
	t.Cleanup(server.Close)

	response := request(t, server, http.MethodGet, "/api/learning-targets/active-coarse-unit-ids", "", false)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if activeTargets.called {
		t.Fatalf("active target usecase should not be called without principal")
	}
}

func newServer(list *fakeListCollectionsForUserUsecase, activate *fakeActivateUsecase, activeTargets *fakeActiveTargetsUsecase, withAuth bool) *httptest.Server {
	handler := router.New(router.Options{
		UnitCollections: unitcollections.NewHandler(list),
		LearningTargets: learningtargets.NewHandler(activate, activeTargets),
	})
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

type fakeListCollectionsForUserUsecase struct {
	called   bool
	request  apivdto.ListUnitCollectionsRequest
	response apivdto.UnitCollectionsResponse
	err      error
}

func (f *fakeListCollectionsForUserUsecase) Execute(_ context.Context, request apivdto.ListUnitCollectionsRequest) (apivdto.UnitCollectionsResponse, error) {
	f.called = true
	f.request = request
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

type fakeActiveTargetsUsecase struct {
	called   bool
	request  learningdto.GetActiveLearningTargetCoarseUnitIDsRequest
	response learningdto.GetActiveLearningTargetCoarseUnitIDsResponse
	err      error
}

func (f *fakeActiveTargetsUsecase) Execute(_ context.Context, request learningdto.GetActiveLearningTargetCoarseUnitIDsRequest) (learningdto.GetActiveLearningTargetCoarseUnitIDsResponse, error) {
	f.called = true
	f.request = request
	return f.response, f.err
}

const userID = "11111111-1111-4111-8111-111111111111"
