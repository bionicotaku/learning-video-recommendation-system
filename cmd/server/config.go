package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type config struct {
	Addr                string
	DatabaseURL         string
	TrustedUserIDHeader string
	PublicAssetBaseURL  string
}

func loadConfig() (config, error) {
	_ = godotenv.Load()
	return loadConfigFromEnv(os.Getenv)
}

func loadConfigFromEnv(getenv func(string) string) (config, error) {
	databaseURL := strings.TrimSpace(getenv("DATABASE_URL"))
	if databaseURL == "" {
		return config{}, fmt.Errorf("DATABASE_URL is not set")
	}

	trustedUserIDHeader := strings.TrimSpace(getenv("API_TRUSTED_USER_ID_HEADER"))
	if trustedUserIDHeader == "" {
		return config{}, fmt.Errorf("API_TRUSTED_USER_ID_HEADER is not set")
	}

	publicAssetBaseURL := strings.TrimRight(strings.TrimSpace(getenv("PUBLIC_ASSET_BASE_URL")), "/")
	if publicAssetBaseURL == "" {
		return config{}, fmt.Errorf("PUBLIC_ASSET_BASE_URL is not set")
	}
	parsedPublicAssetBaseURL, err := url.Parse(publicAssetBaseURL)
	if err != nil || !parsedPublicAssetBaseURL.IsAbs() || (parsedPublicAssetBaseURL.Scheme != "http" && parsedPublicAssetBaseURL.Scheme != "https") {
		return config{}, fmt.Errorf("PUBLIC_ASSET_BASE_URL must be an absolute http(s) URL")
	}

	addr := strings.TrimSpace(getenv("API_ADDR"))
	if addr == "" {
		addr = ":8080"
	}

	return config{
		Addr:                addr,
		DatabaseURL:         databaseURL,
		TrustedUserIDHeader: trustedUserIDHeader,
		PublicAssetBaseURL:  publicAssetBaseURL,
	}, nil
}
