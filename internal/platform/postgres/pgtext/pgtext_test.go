package pgtext_test

import (
	"testing"

	"learning-video-recommendation-system/internal/platform/postgres/pgtext"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestFromStringMapsEmptyToInvalidText(t *testing.T) {
	got := pgtext.FromString("")

	if got.Valid {
		t.Fatalf("FromString(empty).Valid = true, want false")
	}
}

func TestFromStringMapsStringToValidText(t *testing.T) {
	got := pgtext.FromString("example")

	if !got.Valid {
		t.Fatalf("FromString().Valid = false, want true")
	}
	if got.String != "example" {
		t.Fatalf("FromString().String = %q, want example", got.String)
	}
}

func TestToStringMapsInvalidToEmpty(t *testing.T) {
	got := pgtext.ToString(pgtype.Text{})

	if got != "" {
		t.Fatalf("ToString(invalid) = %q, want empty", got)
	}
}

func TestToStringMapsValidTextToString(t *testing.T) {
	got := pgtext.ToString(pgtype.Text{String: "example", Valid: true})

	if got != "example" {
		t.Fatalf("ToString() = %q, want example", got)
	}
}
