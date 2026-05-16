package pguuid_test

import (
	"testing"

	"learning-video-recommendation-system/internal/platform/postgres/pguuid"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestFromStringMapsEmptyToInvalidUUID(t *testing.T) {
	got, err := pguuid.FromString("")
	if err != nil {
		t.Fatalf("FromString() error = %v", err)
	}
	if got.Valid {
		t.Fatalf("FromString(empty).Valid = true, want false")
	}
}

func TestFromStringParsesUUID(t *testing.T) {
	got, err := pguuid.FromString("11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("FromString() error = %v", err)
	}
	if !got.Valid {
		t.Fatalf("FromString().Valid = false, want true")
	}
	if got.String() != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("FromString() = %q", got.String())
	}
}

func TestFromStringRejectsInvalidUUID(t *testing.T) {
	_, err := pguuid.FromString("not-a-uuid")
	if err == nil {
		t.Fatalf("FromString() error = nil, want invalid UUID error")
	}
}

func TestToStringMapsInvalidToEmpty(t *testing.T) {
	got := pguuid.ToString(pgtype.UUID{})
	if got != "" {
		t.Fatalf("ToString(invalid) = %q, want empty", got)
	}
}

func TestToStringReturnsUUIDString(t *testing.T) {
	value, err := pguuid.FromString("11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("FromString() error = %v", err)
	}

	got := pguuid.ToString(value)

	if got != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("ToString() = %q", got)
	}
}

func TestSliceFromStringsParsesUUIDs(t *testing.T) {
	got, err := pguuid.SliceFromStrings([]string{
		"11111111-1111-1111-1111-111111111111",
		"22222222-2222-2222-2222-222222222222",
	})
	if err != nil {
		t.Fatalf("SliceFromStrings() error = %v", err)
	}
	if len(got) != 2 || !got[0].Valid || !got[1].Valid {
		t.Fatalf("SliceFromStrings() = %+v, want 2 valid UUIDs", got)
	}
}
