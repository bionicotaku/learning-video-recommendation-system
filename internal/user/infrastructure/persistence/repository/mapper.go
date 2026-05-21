package repository

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func stringToUUID(value string) (pgtype.UUID, error) {
	var uuid pgtype.UUID
	if err := uuid.Scan(value); err != nil {
		return pgtype.UUID{}, fmt.Errorf("map uuid: %w", err)
	}
	return uuid, nil
}

func uuidToString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return value.String()
}

func textPointer(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	text := value.String
	return &text
}

func textValue(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}

func timestamptzPointer(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time.UTC()
	return &t
}

func timestamptzValue(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

func dateValue(value time.Time) pgtype.Date {
	return pgtype.Date{Time: time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC), Valid: true}
}
