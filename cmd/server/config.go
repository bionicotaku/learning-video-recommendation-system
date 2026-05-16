package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type config struct {
	Addr                string
	DatabaseURL         string
	TrustedUserIDHeader string
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

	addr := strings.TrimSpace(getenv("API_ADDR"))
	if addr == "" {
		addr = ":8080"
	}

	return config{
		Addr:                addr,
		DatabaseURL:         databaseURL,
		TrustedUserIDHeader: trustedUserIDHeader,
	}, nil
}
