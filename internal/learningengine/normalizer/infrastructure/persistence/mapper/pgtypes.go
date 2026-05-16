package mapper

import (
	"time"

	"learning-video-recommendation-system/internal/platform/postgres/pgtext"
	"learning-video-recommendation-system/internal/platform/postgres/pgtime"
	"learning-video-recommendation-system/internal/platform/postgres/pguuid"

	"github.com/jackc/pgx/v5/pgtype"
)

func StringToUUID(value string) (pgtype.UUID, error) {
	return pguuid.FromString(value)
}

func StringsToUUIDs(values []string) ([]pgtype.UUID, error) {
	return pguuid.SliceFromStrings(values)
}

func UUIDToString(value pgtype.UUID) string {
	return pguuid.ToString(value)
}

func TextToString(value pgtype.Text) string {
	return pgtext.ToString(value)
}

func Int32PointerFromPG(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	result := value.Int32
	return &result
}

func Int64FromPG(value pgtype.Int8) int64 {
	if !value.Valid {
		return 0
	}
	return value.Int64
}

func TimePointerToPG(value *time.Time) pgtype.Timestamptz {
	return pgtime.ToTimestamptz(value)
}

func TimeFromPG(value pgtype.Timestamptz) time.Time {
	return pgtime.FromTimestamptz(value)
}
