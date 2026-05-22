package legaldocuments_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	legaldocumentshandler "learning-video-recommendation-system/internal/api/infrastructure/http/handler/legaldocuments"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/router"
	userdto "learning-video-recommendation-system/internal/user/application/dto"
	userrepo "learning-video-recommendation-system/internal/user/application/repository"
	userservice "learning-video-recommendation-system/internal/user/application/service"
)

func TestGetLegalDocumentAllowsAnonymousRequest(t *testing.T) {
	service := &fakeGetLegalDocumentUsecase{response: userdto.GetLegalDocumentResponse{
		Type:      "privacy-policy",
		Title:     "隐私政策",
		Markdown:  "# 隐私政策\n",
		UpdatedAt: strPtr("2026-05-21T00:00:00Z"),
		Version:   strPtr("2026-05-21"),
	}}
	server := newServer(service)
	t.Cleanup(server.Close)

	response := getLegalDocument(t, server, "/api/legal-documents/privacy-policy", "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if !service.called || service.request.Type != "privacy-policy" {
		t.Fatalf("service request not mapped: %+v", service.request)
	}

	var payload userdto.GetLegalDocumentResponse
	if err := json.Unmarshal([]byte(readBody(t, response)), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Type != "privacy-policy" || payload.Title != "隐私政策" || payload.Markdown == "" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestGetLegalDocumentIgnoresMalformedGatewayUserinfo(t *testing.T) {
	service := &fakeGetLegalDocumentUsecase{response: userdto.GetLegalDocumentResponse{
		Type:     "user-agreement",
		Title:    "用户协议",
		Markdown: "# 用户协议\n",
	}}
	server := newServer(service)
	t.Cleanup(server.Close)

	response := getLegalDocument(t, server, "/api/legal-documents/user-agreement", "not-base64")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if !service.called || service.request.Type != "user-agreement" {
		t.Fatalf("service request not mapped: %+v", service.request)
	}
}

func TestGetLegalDocumentMapsUnsupportedTypeToInvalidRequest(t *testing.T) {
	service := &fakeGetLegalDocumentUsecase{err: userservice.ValidationError("unsupported legal document type")}
	server := newServer(service)
	t.Cleanup(server.Close)

	response := getLegalDocument(t, server, "/api/legal-documents/terms", "")
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
	}
	body := readBody(t, response)
	if !strings.Contains(body, `"invalid_request"`) {
		t.Fatalf("expected invalid_request error: %s", body)
	}
}

func TestGetLegalDocumentMapsMissingConfiguredDocumentToInternalError(t *testing.T) {
	service := &fakeGetLegalDocumentUsecase{err: userrepo.ErrLegalDocumentNotFound}
	server := newServer(service)
	t.Cleanup(server.Close)

	response := getLegalDocument(t, server, "/api/legal-documents/privacy-policy", "")
	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", response.StatusCode, readBody(t, response))
	}
	body := readBody(t, response)
	if !strings.Contains(body, `"internal_error"`) {
		t.Fatalf("expected internal_error: %s", body)
	}
}

func newServer(service *fakeGetLegalDocumentUsecase) *httptest.Server {
	handler := router.New(router.Options{LegalDocuments: legaldocumentshandler.NewHandler(service)})
	handler = auth.PrincipalMiddleware(auth.Options{GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo"})(handler)
	handler = middleware.RequestID(handler)
	return httptest.NewServer(handler)
}

func getLegalDocument(t *testing.T, server *httptest.Server, path string, gatewayUserinfo string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if gatewayUserinfo != "" {
		request.Header.Set("X-Apigateway-Api-Userinfo", gatewayUserinfo)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	return response
}

func readBody(t *testing.T, response *http.Response) string {
	t.Helper()
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(body)
}

func strPtr(value string) *string {
	return &value
}

type fakeGetLegalDocumentUsecase struct {
	called   bool
	request  userdto.GetLegalDocumentRequest
	response userdto.GetLegalDocumentResponse
	err      error
}

func (f *fakeGetLegalDocumentUsecase) Execute(_ context.Context, request userdto.GetLegalDocumentRequest) (userdto.GetLegalDocumentResponse, error) {
	f.called = true
	f.request = request
	return f.response, f.err
}
