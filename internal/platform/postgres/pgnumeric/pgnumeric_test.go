package pgnumeric_test

import (
	"testing"

	"learning-video-recommendation-system/internal/platform/postgres/pgnumeric"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestFromFloat64MapsToNumeric(t *testing.T) {
	got, err := pgnumeric.FromFloat64(1.25)
	if err != nil {
		t.Fatalf("FromFloat64() error = %v", err)
	}

	value, err := pgnumeric.ToFloat64(got)
	if err != nil {
		t.Fatalf("ToFloat64() error = %v", err)
	}
	if value != 1.25 {
		t.Fatalf("ToFloat64(FromFloat64()) = %v, want 1.25", value)
	}
}

func TestToFloat64MapsInvalidToZero(t *testing.T) {
	got, err := pgnumeric.ToFloat64(pgtype.Numeric{})
	if err != nil {
		t.Fatalf("ToFloat64() error = %v", err)
	}
	if got != 0 {
		t.Fatalf("ToFloat64(invalid) = %v, want 0", got)
	}
}
