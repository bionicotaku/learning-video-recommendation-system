package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
)

func TestRequirePrincipalRejectsMissingPrincipal(t *testing.T) {
	_, err := auth.RequirePrincipal(context.Background())

	if err == nil {
		t.Fatalf("expected missing principal to be rejected")
	}
}

func TestRequirePrincipalReturnsTrustedPrincipal(t *testing.T) {
	ctx := auth.WithPrincipal(context.Background(), auth.Principal{UserID: "user-1"})

	principal, err := auth.RequirePrincipal(ctx)
	if err != nil {
		t.Fatalf("expected principal: %v", err)
	}
	if principal.UserID != "user-1" {
		t.Fatalf("unexpected user id: %s", principal.UserID)
	}
}

func TestTrustedHeaderPrincipalMiddlewareInjectsPrincipal(t *testing.T) {
	var got auth.Principal
	var ok bool
	handler := auth.TrustedHeaderPrincipalMiddleware("X-Trusted-User-ID")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok = auth.PrincipalFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Trusted-User-ID", " user-1 ")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}
	if !ok || got.UserID != "user-1" {
		t.Fatalf("expected trusted principal, got %#v ok=%v", got, ok)
	}
}
