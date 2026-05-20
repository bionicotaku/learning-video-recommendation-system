package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
)

const defaultGatewayUserinfoHeader = "X-Apigateway-Api-Userinfo"

type Principal struct {
	UserID string
}

type Options struct {
	DevMode               bool
	GatewayUserinfoHeader string
}

type principalContextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok && principal.UserID != ""
}

func RequirePrincipal(ctx context.Context) (Principal, error) {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		return Principal{}, ErrMissingPrincipal
	}
	return principal, nil
}

func FakePrincipalMiddleware(principal Principal) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
		})
	}
}

func PrincipalMiddleware(options Options) func(http.Handler) http.Handler {
	headerName := strings.TrimSpace(options.GatewayUserinfoHeader)
	if headerName == "" {
		headerName = defaultGatewayUserinfoHeader
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userinfo := strings.TrimSpace(r.Header.Get(headerName))
			if userinfo != "" {
				if principal, ok := principalFromPayload(userinfo); ok {
					next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			if options.DevMode {
				if principal, ok := principalFromAuthorization(r.Header.Get("Authorization")); ok {
					next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func principalFromAuthorization(authorization string) (Principal, bool) {
	const bearerPrefix = "Bearer "
	authorization = strings.TrimSpace(authorization)
	if !strings.HasPrefix(authorization, bearerPrefix) {
		return Principal{}, false
	}
	token := strings.TrimSpace(strings.TrimPrefix(authorization, bearerPrefix))
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return Principal{}, false
	}
	return principalFromPayload(parts[1])
}

func principalFromPayload(encodedPayload string) (Principal, bool) {
	payload, err := decodeBase64URL(strings.TrimSpace(encodedPayload))
	if err != nil {
		return Principal{}, false
	}

	var claims struct {
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Principal{}, false
	}
	userID := strings.TrimSpace(claims.Sub)
	if userID == "" {
		return Principal{}, false
	}
	return Principal{UserID: userID}, true
}

func decodeBase64URL(value string) ([]byte, error) {
	if decoded, err := base64.RawURLEncoding.DecodeString(value); err == nil {
		return decoded, nil
	}
	return base64.URLEncoding.DecodeString(value)
}
