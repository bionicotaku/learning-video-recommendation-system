package infrastructure_test

import (
	"testing"

	infra "learning-video-recommendation-system/internal/learningengine/infrastructure"
)

func TestConfigValidateRequiresDatabaseURL(t *testing.T) {
	cfg := infra.Config{}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error")
	}

	if got, want := err.Error(), "DATABASE_URL must be set for learning engine PostgreSQL direct access"; got != want {
		t.Fatalf("Validate() error = %q, want %q", got, want)
	}
}

func TestConfigValidateDoesNotFallbackToSupabaseURL(t *testing.T) {
	cfg := infra.Config{SupabaseURL: "https://example.supabase.co"}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error")
	}

	if got, want := err.Error(), "DATABASE_URL must be set for direct PostgreSQL access; SUPABASE_URL cannot be used as a fallback"; got != want {
		t.Fatalf("Validate() error = %q, want %q", got, want)
	}
}
