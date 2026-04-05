package infrastructure

import "testing"

func TestConfigValidateRequiresDatabaseURL(t *testing.T) {
	cfg := Config{}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error")
	}

	if got, want := err.Error(), "DATABASE_URL must be set for recommendation PostgreSQL direct access"; got != want {
		t.Fatalf("Validate() error = %q, want %q", got, want)
	}
}

func TestConfigValidateDoesNotFallbackToSupabaseURL(t *testing.T) {
	cfg := Config{SupabaseURL: "https://example.supabase.co"}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error")
	}

	if got, want := err.Error(), "DATABASE_URL must be set for direct PostgreSQL access; SUPABASE_URL cannot be used as a fallback"; got != want {
		t.Fatalf("Validate() error = %q, want %q", got, want)
	}
}
