package feedback_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
	feedbackhandler "learning-video-recommendation-system/internal/api/infrastructure/http/handler/feedback"
	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
	"learning-video-recommendation-system/internal/api/infrastructure/http/router"
	userdto "learning-video-recommendation-system/internal/user/application/dto"
)

func TestSubmitFeedbackAcceptsPayloadAndImages(t *testing.T) {
	service := &fakeSubmitFeedbackUsecase{response: userdto.SubmitFeedbackResponse{
		FeedbackID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		Accepted:   true,
		ImageCount: 2,
		CreatedAt:  "2026-05-22T18:30:00Z",
	}}
	server := newServer(service, true)
	t.Cleanup(server.Close)

	body, contentType := multipartBody(t, `{"message":"bug"}`, "11111111-1111-4111-8111-111111111111", [][]byte{jpegBytes(t), jpegBytes(t)})
	response := postMultipart(t, server, body, contentType, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if service.request.UserID != "user-1" || service.request.ClientFeedbackID == nil || len(service.request.Images) != 2 {
		t.Fatalf("request not mapped: %+v", service.request)
	}
	if service.request.Images[0].SortOrder != 1 || service.request.Images[1].SortOrder != 2 {
		t.Fatalf("sort order not mapped: %+v", service.request.Images)
	}
	if service.request.Images[0].ContentType != "image/jpeg" || service.request.Images[0].Width != 1 || service.request.Images[0].Height != 1 {
		t.Fatalf("image metadata not mapped: %+v", service.request.Images[0])
	}

	responseBody := readBody(t, response)
	var payload userdto.SubmitFeedbackResponse
	if err := json.Unmarshal([]byte(responseBody), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.FeedbackID != "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" || payload.ImageCount != 2 || !payload.Accepted {
		t.Fatalf("unexpected response: %+v", payload)
	}
}

func TestSubmitFeedbackRejectsInvalidPayloadAndImages(t *testing.T) {
	cases := []struct {
		name        string
		payload     string
		images      [][]byte
		contentType string
	}{
		{name: "non object payload", payload: `[]`, images: nil},
		{name: "too many images", payload: `{}`, images: [][]byte{jpegBytes(t), jpegBytes(t), jpegBytes(t), jpegBytes(t), jpegBytes(t), jpegBytes(t)}},
		{name: "bad image bytes", payload: `{}`, images: [][]byte{[]byte("not-jpeg")}, contentType: "image/jpeg"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakeSubmitFeedbackUsecase{}
			server := newServer(service, true)
			t.Cleanup(server.Close)

			body, contentType := multipartBody(t, tc.payload, "", tc.images)
			response := postMultipart(t, server, body, contentType, true)
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
			}
			if service.called {
				t.Fatalf("service should not be called")
			}
		})
	}
}

func TestSubmitFeedbackMapsOversizeRequestToPayloadTooLarge(t *testing.T) {
	server := newServer(&fakeSubmitFeedbackUsecase{}, true)
	t.Cleanup(server.Close)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("payload", `{"message":"bug"}`); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	part, err := createJPEGPart(writer)
	if err != nil {
		t.Fatalf("create image: %v", err)
	}
	if _, err := part.Write(bytes.Repeat([]byte{'x'}, 5*1024*1024)); err != nil {
		t.Fatalf("write image: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}

	response := postMultipart(t, server, body.Bytes(), writer.FormDataContentType(), true)
	if response.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d: %s", response.StatusCode, readBody(t, response))
	}
	payload := readBody(t, response)
	if !strings.Contains(payload, `"payload_too_large"`) {
		t.Fatalf("expected payload_too_large error: %s", payload)
	}
}

func TestSubmitFeedbackRequiresPrincipal(t *testing.T) {
	service := &fakeSubmitFeedbackUsecase{}
	server := newServer(service, true)
	t.Cleanup(server.Close)

	body, contentType := multipartBody(t, `{}`, "", nil)
	response := postMultipart(t, server, body, contentType, false)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if service.called {
		t.Fatalf("service should not be called")
	}
}

func newServer(service *fakeSubmitFeedbackUsecase, withAuth bool) *httptest.Server {
	handler := router.New(router.Options{Feedback: feedbackhandler.NewHandler(service)})
	handler = middleware.BodyLimitByPath(1<<20, map[string]int64{"/api/feedback": 5 << 20})(handler)
	if withAuth {
		handler = auth.PrincipalMiddleware(auth.Options{GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo"})(handler)
	}
	handler = middleware.RequestID(handler)
	return httptest.NewServer(handler)
}

func multipartBody(t *testing.T, payload string, clientFeedbackID string, images [][]byte) ([]byte, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("payload", payload); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	if clientFeedbackID != "" {
		if err := writer.WriteField("client_feedback_id", clientFeedbackID); err != nil {
			t.Fatalf("write client_feedback_id: %v", err)
		}
	}
	for _, image := range images {
		part, err := createJPEGPart(writer)
		if err != nil {
			t.Fatalf("create image: %v", err)
		}
		if _, err := part.Write(image); err != nil {
			t.Fatalf("write image: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}
	return body.Bytes(), writer.FormDataContentType()
}

func createJPEGPart(writer *multipart.Writer) (io.Writer, error) {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="images"; filename="screenshot.jpg"`)
	header.Set("Content-Type", "image/jpeg")
	return writer.CreatePart(header)
}

func postMultipart(t *testing.T, server *httptest.Server, body []byte, contentType string, setPrincipal bool) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/feedback", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Content-Type", contentType)
	if setPrincipal {
		request.Header.Set("X-Apigateway-Api-Userinfo", "eyJzdWIiOiJ1c2VyLTEifQ")
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post: %v", err)
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

func jpegBytes(t *testing.T) []byte {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString("/9j/2wBDAAIBAQEBAQIBAQECAgICAgQDAgICAgUEBAMEBgUGBgYFBgYGBwkIBwcHBgYGCAsICQoKCgoKBggLDAsKDAkKCgr/2wBDAQICAgICAgQDAwQKCAYGCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgr/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAX/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIQAxAAAAH/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/9oACAEBAAEFAqf/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oACAEDAQE/Aaf/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oACAECAQE/Aaf/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/9oACAEBAAY/Aqf/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/9oACAEBAAE/IV//2gAMAwEAAgADAAAAEP/EABQRAQAAAAAAAAAAAAAAAAAAABD/2gAIAQMBAT8Qf//EABQRAQAAAAAAAAAAAAAAAAAAABD/2gAIAQIBAT8Qf//EABQQAQAAAAAAAAAAAAAAAAAAABD/2gAIAQEAAT8Qf//Z")
	if err != nil {
		t.Fatalf("decode jpeg: %v", err)
	}
	return raw
}

type fakeSubmitFeedbackUsecase struct {
	called   bool
	request  userdto.SubmitFeedbackRequest
	response userdto.SubmitFeedbackResponse
	err      error
}

func (f *fakeSubmitFeedbackUsecase) Execute(_ context.Context, request userdto.SubmitFeedbackRequest) (userdto.SubmitFeedbackResponse, error) {
	f.called = true
	f.request = request
	if f.response.CreatedAt == "" {
		f.response = userdto.SubmitFeedbackResponse{
			FeedbackID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
			Accepted:   true,
			ImageCount: int32(len(request.Images)),
			CreatedAt:  time.Date(2026, 5, 22, 18, 30, 0, 0, time.UTC).Format(time.RFC3339),
		}
	}
	return f.response, f.err
}
