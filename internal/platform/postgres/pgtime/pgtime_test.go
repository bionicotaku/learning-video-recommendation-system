package pgtime_test

import (
	"testing"
	"time"

	"learning-video-recommendation-system/internal/platform/postgres/pgtime"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestToTimestamptzMapsNilToInvalid(t *testing.T) {
	got := pgtime.ToTimestamptz(nil)

	if got.Valid {
		t.Fatalf("ToTimestamptz(nil).Valid = true, want false")
	}
}

func TestToTimestamptzMapsZeroTimeToInvalid(t *testing.T) {
	zero := time.Time{}

	got := pgtime.ToTimestamptz(&zero)

	if got.Valid {
		t.Fatalf("ToTimestamptz(zero).Valid = true, want false")
	}
}

func TestToTimestamptzNormalizesToUTC(t *testing.T) {
	localTime := time.Date(2026, 5, 15, 10, 0, 0, 0, time.FixedZone("PDT", -7*60*60))

	got := pgtime.ToTimestamptz(&localTime)

	if !got.Valid {
		t.Fatalf("ToTimestamptz().Valid = false, want true")
	}
	if got.Time.Location() != time.UTC {
		t.Fatalf("ToTimestamptz().Time location = %v, want UTC", got.Time.Location())
	}
	if !got.Time.Equal(localTime) {
		t.Fatalf("ToTimestamptz().Time = %v, want same instant as %v", got.Time, localTime)
	}
}

func TestFromTimestamptzMapsInvalidToZeroTime(t *testing.T) {
	got := pgtime.FromTimestamptz(pgtype.Timestamptz{})

	if !got.IsZero() {
		t.Fatalf("FromTimestamptz(invalid) = %v, want zero", got)
	}
}

func TestFromTimestamptzNormalizesToUTC(t *testing.T) {
	localTime := time.Date(2026, 5, 15, 10, 0, 0, 0, time.FixedZone("PDT", -7*60*60))

	got := pgtime.FromTimestamptz(pgtype.Timestamptz{Time: localTime, Valid: true})

	if got.Location() != time.UTC {
		t.Fatalf("FromTimestamptz() location = %v, want UTC", got.Location())
	}
	if !got.Equal(localTime) {
		t.Fatalf("FromTimestamptz() = %v, want same instant as %v", got, localTime)
	}
}

func TestPtrFromTimestamptzMapsInvalidToNil(t *testing.T) {
	got := pgtime.PtrFromTimestamptz(pgtype.Timestamptz{})

	if got != nil {
		t.Fatalf("PtrFromTimestamptz(invalid) = %v, want nil", *got)
	}
}

func TestPtrFromTimestamptzNormalizesToUTC(t *testing.T) {
	localTime := time.Date(2026, 5, 15, 10, 0, 0, 0, time.FixedZone("PDT", -7*60*60))

	got := pgtime.PtrFromTimestamptz(pgtype.Timestamptz{Time: localTime, Valid: true})

	if got == nil {
		t.Fatalf("PtrFromTimestamptz() = nil, want value")
	}
	if got.Location() != time.UTC {
		t.Fatalf("PtrFromTimestamptz() location = %v, want UTC", got.Location())
	}
	if !got.Equal(localTime) {
		t.Fatalf("PtrFromTimestamptz() = %v, want same instant as %v", *got, localTime)
	}
}
