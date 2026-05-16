package main

import "testing"

func TestLoadConfigFromEnvRequiresTrustedUserIDHeader(t *testing.T) {
	_, err := loadConfigFromEnv(func(key string) string {
		switch key {
		case "DATABASE_URL":
			return "postgres://example"
		default:
			return ""
		}
	})

	if err == nil {
		t.Fatalf("expected missing trusted user id header to fail")
	}
}

func TestLoadConfigFromEnvReadsTrustedUserIDHeader(t *testing.T) {
	config, err := loadConfigFromEnv(func(key string) string {
		switch key {
		case "DATABASE_URL":
			return "postgres://example"
		case "API_TRUSTED_USER_ID_HEADER":
			return "X-Trusted-User-ID"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if config.Addr != ":8080" {
		t.Fatalf("unexpected default addr: %s", config.Addr)
	}
	if config.TrustedUserIDHeader != "X-Trusted-User-ID" {
		t.Fatalf("unexpected trusted header: %s", config.TrustedUserIDHeader)
	}
}

func TestBuildHTTPHandlerRequiresTrustedUserIDHeader(t *testing.T) {
	_, err := buildHTTPHandler(nil, nil, config{})

	if err == nil {
		t.Fatalf("expected missing trusted user id header to fail")
	}
}
