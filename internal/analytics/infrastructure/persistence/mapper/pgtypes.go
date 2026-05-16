package mapper

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func StringToUUID(value string) (pgtype.UUID, error) {
	if value == "" {
		return pgtype.UUID{}, nil
	}
	var uuid pgtype.UUID
	if err := uuid.Scan(value); err != nil {
		return pgtype.UUID{}, err
	}
	return uuid, nil
}

func UUIDToString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return value.String()
}

func StringToText(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
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
	if value == nil || value.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}
