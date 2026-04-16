package mapper

import (
	"strconv"
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

func StringToText(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func TextToString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
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
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func TimePointerFromPG(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
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
	var result pgtype.Numeric
	if err := result.Scan(strconv.FormatFloat(value, 'f', -1, 64)); err != nil {
		return pgtype.Numeric{}, err
	}
	return result, nil
}

func NumericToFloat64(value pgtype.Numeric) (float64, error) {
	floatValue, err := value.Float64Value()
	if err != nil {
		return 0, err
	}
	if !floatValue.Valid {
		return 0, nil
	}
	return floatValue.Float64, nil
}
