package mapper

import (
	"time"

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

func TimePointerToPG(value *time.Time) pgtype.Timestamptz {
	return pgtime.ToTimestamptz(value)
}

func TimeFromPG(value pgtype.Timestamptz) time.Time {
	return pgtime.FromTimestamptz(value)
}
