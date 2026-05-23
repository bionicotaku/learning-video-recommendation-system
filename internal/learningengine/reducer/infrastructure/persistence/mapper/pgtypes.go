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

func UUIDsToStrings(values []pgtype.UUID) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if !value.Valid {
			continue
		}
		result = append(result, pguuid.ToString(value))
	}
	return result
}

func StringToText(value string) pgtype.Text {
	return pgtext.FromString(value)
}

func TextToString(value pgtype.Text) string {
	return pgtext.ToString(value)
}

func BoolPointerToPG(value *bool) pgtype.Bool {
	if value == nil {
		return pgtype.Bool{}
	}
	return pgtype.Bool{Bool: *value, Valid: true}
}

func Int16PointerToPG(value *int16) pgtype.Int2 {
	if value == nil {
		return pgtype.Int2{}
	}
	return pgtype.Int2{Int16: *value, Valid: true}
}

func Int32PointerToPG(value *int32) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *value, Valid: true}
}

func Int32PointerFromPG(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	result := value.Int32
	return &result
}

func TimePointerToPG(value *time.Time) pgtype.Timestamptz {
	return pgtime.ToTimestamptz(value)
}

func TimePointerFromPG(value pgtype.Timestamptz) *time.Time {
	return pgtime.PtrFromTimestamptz(value)
}

func TimeFromPG(value pgtype.Timestamptz) time.Time {
	return pgtime.FromTimestamptz(value)
}

func Int16PointerFromPG(value pgtype.Int2) *int16 {
	if !value.Valid {
		return nil
	}
	result := value.Int16
	return &result
}

func BoolPointerFromPG(value pgtype.Bool) *bool {
	if !value.Valid {
		return nil
	}
	result := value.Bool
	return &result
}

func Float64ToNumeric(value float64) (pgtype.Numeric, error) {
	return pgnumeric.FromFloat64(value)
}

func NumericToFloat64(value pgtype.Numeric) (float64, error) {
	return pgnumeric.ToFloat64(value)
}
