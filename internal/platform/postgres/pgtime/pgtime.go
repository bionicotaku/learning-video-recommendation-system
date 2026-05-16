package pgtime

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ToTimestamptz maps a Go time pointer to a Postgres timestamptz.
func ToTimestamptz(value *time.Time) pgtype.Timestamptz {
	if value == nil || value.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

// FromTimestamptz maps a Postgres timestamptz to a UTC Go time.
func FromTimestamptz(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time.UTC()
}

// PtrFromTimestamptz maps a Postgres timestamptz to a UTC Go time pointer.
func PtrFromTimestamptz(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time.UTC()
	return &result
}
