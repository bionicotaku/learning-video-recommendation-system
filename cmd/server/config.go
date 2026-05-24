package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const defaultAPIGatewayUserinfoHeader = "X-Apigateway-Api-Userinfo"
const defaultPGMaxConns = 5

type config struct {
	Addr                     string
	DatabaseURL              string
	PublicAssetBaseURL       string
	DevMode                  bool
	APIGatewayUserinfoHeader string
	PGMaxConns               int
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

	devMode := false
	if rawDevMode := strings.TrimSpace(getenv("DEV_MODE")); rawDevMode != "" {
		parsed, err := strconv.ParseBool(rawDevMode)
		if err != nil {
			return config{}, fmt.Errorf("DEV_MODE must be a boolean")
		}
		devMode = parsed
	}

	gatewayUserinfoHeader := strings.TrimSpace(getenv("API_GATEWAY_USERINFO_HEADER"))
	if gatewayUserinfoHeader == "" {
		gatewayUserinfoHeader = defaultAPIGatewayUserinfoHeader
	}

	pgMaxConns := defaultPGMaxConns
	if rawPGMaxConns := strings.TrimSpace(getenv("PG_MAX_CONNS")); rawPGMaxConns != "" {
		parsed, err := strconv.ParseInt(rawPGMaxConns, 10, 32)
		if err != nil || parsed < 1 {
			return config{}, fmt.Errorf("PG_MAX_CONNS must be a positive integer")
		}
		pgMaxConns = int(parsed)
	}

	return config{
		Addr:                     addr,
		DatabaseURL:              databaseURL,
		PublicAssetBaseURL:       publicAssetBaseURL,
		DevMode:                  devMode,
		APIGatewayUserinfoHeader: gatewayUserinfoHeader,
		PGMaxConns:               pgMaxConns,
	}, nil
}
