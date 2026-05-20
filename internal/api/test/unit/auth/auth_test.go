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

func TestPrincipalMiddlewareInjectsGatewayUserinfoPrincipal(t *testing.T) {
	var got auth.Principal
	var ok bool
	handler := auth.PrincipalMiddleware(auth.Options{GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok = auth.PrincipalFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Apigateway-Api-Userinfo", " eyJzdWIiOiJ1c2VyLTEifQ ")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}
	if !ok || got.UserID != "user-1" {
		t.Fatalf("expected trusted principal, got %#v ok=%v", got, ok)
	}
}

func TestPrincipalMiddlewareFallsBackToAuthorizationInDevMode(t *testing.T) {
	var got auth.Principal
	var ok bool
	handler := auth.PrincipalMiddleware(auth.Options{
		DevMode:               true,
		GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo",
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok = auth.PrincipalFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer e30.eyJzdWIiOiJ1c2VyLWRldiJ9.sig")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}
	if !ok || got.UserID != "user-dev" {
		t.Fatalf("expected dev principal, got %#v ok=%v", got, ok)
	}
}

func TestPrincipalMiddlewareIgnoresAuthorizationOutsideDevMode(t *testing.T) {
	var ok bool
	handler := auth.PrincipalMiddleware(auth.Options{GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok = auth.PrincipalFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer e30.eyJzdWIiOiJ1c2VyLWRldiJ9.sig")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if ok {
		t.Fatalf("expected Authorization to be ignored when DEV_MODE is false")
	}
}

func TestPrincipalMiddlewareDoesNotFallbackWhenGatewayHeaderIsMalformed(t *testing.T) {
	var ok bool
	handler := auth.PrincipalMiddleware(auth.Options{
		DevMode:               true,
		GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo",
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok = auth.PrincipalFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Apigateway-Api-Userinfo", "not-base64")
	request.Header.Set("Authorization", "Bearer e30.eyJzdWIiOiJ1c2VyLWRldiJ9.sig")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if ok {
		t.Fatalf("expected malformed gateway header to block dev fallback")
	}
}

func TestPrincipalMiddlewareRejectsMissingSubClaim(t *testing.T) {
	var ok bool
	handler := auth.PrincipalMiddleware(auth.Options{GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok = auth.PrincipalFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Apigateway-Api-Userinfo", "eyJlbWFpbCI6InVAZXhhbXBsZS5jb20ifQ")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if ok {
		t.Fatalf("expected missing sub claim to be rejected")
	}
}

func TestPrincipalMiddlewareRejectsMalformedAuthorizationFallback(t *testing.T) {
	var ok bool
	handler := auth.PrincipalMiddleware(auth.Options{
		DevMode:               true,
		GatewayUserinfoHeader: "X-Apigateway-Api-Userinfo",
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok = auth.PrincipalFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	for _, authorization := range []string{"Basic token", "Bearer malformed", "Bearer e30.not-base64.sig"} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Authorization", authorization)
		recorder := httptest.NewRecorder()
		ok = false

		handler.ServeHTTP(recorder, request)

		if ok {
			t.Fatalf("expected %q to be rejected", authorization)
		}
	}
}
