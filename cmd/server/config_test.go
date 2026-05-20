package main

import "testing"

func TestLoadConfigFromEnvDefaultsAuthConfig(t *testing.T) {
	config, err := loadConfigFromEnv(func(key string) string {
		switch key {
		case "DATABASE_URL":
			return "postgres://example"
		case "PUBLIC_ASSET_BASE_URL":
			return "https://cdn.example.com"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if config.DevMode {
		t.Fatalf("expected DEV_MODE to default false")
	}
	if config.APIGatewayUserinfoHeader != "X-Apigateway-Api-Userinfo" {
		t.Fatalf("unexpected gateway userinfo header: %s", config.APIGatewayUserinfoHeader)
	}
}

func TestLoadConfigFromEnvReadsOptionalAuthConfig(t *testing.T) {
	config, err := loadConfigFromEnv(func(key string) string {
		switch key {
		case "DATABASE_URL":
			return "postgres://example"
		case "PUBLIC_ASSET_BASE_URL":
			return "https://cdn.example.com/assets/"
		case "DEV_MODE":
			return "true"
		case "API_GATEWAY_USERINFO_HEADER":
			return "X-Custom-Userinfo"
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
	if !config.DevMode {
		t.Fatalf("expected DEV_MODE true")
	}
	if config.APIGatewayUserinfoHeader != "X-Custom-Userinfo" {
		t.Fatalf("unexpected gateway userinfo header: %s", config.APIGatewayUserinfoHeader)
	}
	if config.PublicAssetBaseURL != "https://cdn.example.com/assets" {
		t.Fatalf("unexpected public asset base url: %s", config.PublicAssetBaseURL)
	}
}

func TestLoadConfigFromEnvRejectsInvalidDevMode(t *testing.T) {
	_, err := loadConfigFromEnv(func(key string) string {
		switch key {
		case "DATABASE_URL":
			return "postgres://example"
		case "PUBLIC_ASSET_BASE_URL":
			return "https://cdn.example.com"
		case "DEV_MODE":
			return "sometimes"
		default:
			return ""
		}
	})

	if err == nil {
		t.Fatalf("expected invalid DEV_MODE to fail")
	}
}
