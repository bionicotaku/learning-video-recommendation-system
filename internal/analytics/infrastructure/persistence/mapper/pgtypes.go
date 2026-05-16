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

func UUIDToString(value pgtype.UUID) string {
	return pguuid.ToString(value)
}

func StringToText(value string) pgtype.Text {
	return pgtext.FromString(value)
}

func Int64PointerToPG(value *int64) pgtype.Int8 {
	if value == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *value, Valid: true}
}

func Int32PointerToPG(value *int32) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *value, Valid: true}
}

func TimePointerToPG(value *time.Time) pgtype.Timestamptz {
	return pgtime.ToTimestamptz(value)
}
