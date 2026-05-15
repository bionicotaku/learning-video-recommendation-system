package mapper

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func StringToUUID(value string) (pgtype.UUID, error) {
	if value == "" {
		return pgtype.UUID{}, nil
	}

	var result pgtype.UUID
	if err := result.Scan(value); err != nil {
		return pgtype.UUID{}, err
	}
	return result, nil
}

func UUIDToString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return value.String()
}

func TextToString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
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
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func TimeFromPG(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}
