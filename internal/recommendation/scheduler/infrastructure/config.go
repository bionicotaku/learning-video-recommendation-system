package infrastructure

import (
	"errors"
	"fmt"
	"os"
)

const (
	envDatabaseURL = "DATABASE_URL"
	envSupabaseURL = "SUPABASE_URL"
)

// Config defines runtime configuration used by the recommendation infrastructure.
type Config struct {
	DatabaseURL string
	SupabaseURL string
}

// LoadConfig loads recommendation configuration from environment variables.
func LoadConfig() Config {
	return Config{
		DatabaseURL: os.Getenv(envDatabaseURL),
		SupabaseURL: os.Getenv(envSupabaseURL),
	}
}

// Validate ensures Recommendation is configured to use PostgreSQL direct access.
func (c Config) Validate() error {
	if c.DatabaseURL == "" {
		if c.SupabaseURL != "" {
			return fmt.Errorf("%s must be set for direct PostgreSQL access; %s cannot be used as a fallback", envDatabaseURL, envSupabaseURL)
		}

		return errors.New("DATABASE_URL must be set for recommendation PostgreSQL direct access")
	}

	return nil
}
