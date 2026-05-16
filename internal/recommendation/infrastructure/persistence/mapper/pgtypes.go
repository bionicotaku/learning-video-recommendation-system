package mapper

import (
	"time"

	"learning-video-recommendation-system/internal/platform/postgres/pgnumeric"
	"learning-video-recommendation-system/internal/platform/postgres/pgtext"
	"learning-video-recommendation-system/internal/platform/postgres/pgtime"
	"learning-video-recommendation-system/internal/platform/postgres/pguuid"

	"github.com/jackc/pgx/v5/pgtype"
)

func StringToUUID(value string) (pgtype.UUID, error) {
	return pguuid.FromString(value)
}

func UUIDToString(value pgtype.UUID) string {
	return pguuid.ToString(value)
}

func TextToString(value pgtype.Text) string {
	return pgtext.ToString(value)
}

func StringToText(value string) pgtype.Text {
	return pgtext.FromString(value)
}

func TimePointerFromPG(value pgtype.Timestamptz) *time.Time {
	return pgtime.PtrFromTimestamptz(value)
}

func TimePointerToPG(value *time.Time) pgtype.Timestamptz {
	return pgtime.ToTimestamptz(value)
}

func TimeFromPG(value pgtype.Timestamptz) time.Time {
	return pgtime.FromTimestamptz(value)
}

func Int32PointerFromPG(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	result := value.Int32
	return &result
}

func Int32FromPG(value pgtype.Int4) int32 {
	if !value.Valid {
		return 0
	}
	return value.Int32
}

func Int64PointerFromPG(value pgtype.Int8) *int64 {
	if !value.Valid {
		return nil
	}
	result := value.Int64
	return &result
}

func Int32PointerToPG(value *int32) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *value, Valid: true}
}

func Int64PointerToPG(value *int64) pgtype.Int8 {
	if value == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *value, Valid: true}
}

func Int16PointerFromPG(value pgtype.Int2) *int16 {
	if !value.Valid {
		return nil
	}
	result := value.Int16
	return &result
}

func Float64ToNumeric(value float64) (pgtype.Numeric, error) {
	return pgnumeric.FromFloat64(value)
}

func NumericToFloat64(value pgtype.Numeric) (float64, error) {
	return pgnumeric.ToFloat64(value)
}
