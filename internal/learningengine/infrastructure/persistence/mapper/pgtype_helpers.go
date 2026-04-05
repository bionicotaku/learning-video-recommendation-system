package mapper

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func requiredUUID(value pgtype.UUID, field string) (uuid.UUID, error) {
	if !value.Valid {
		return uuid.Nil, fmt.Errorf("%s is invalid", field)
	}

	return uuid.UUID(value.Bytes), nil
}

func optionalUUID(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}

	id := uuid.UUID(value.Bytes)
	return &id
}

func UUIDToPG(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: id != uuid.Nil}
}

func OptionalUUIDToPG(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}

	return pgtype.UUID{Bytes: [16]byte(*id), Valid: true}
}

func requiredTime(value pgtype.Timestamptz, field string) (time.Time, error) {
	if !value.Valid {
		return time.Time{}, fmt.Errorf("%s is invalid", field)
	}

	return value.Time, nil
}

func optionalTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}

	t := value.Time
	return &t
}

func TimeToPG(value time.Time) pgtype.Timestamptz {
	if value.IsZero() {
		return pgtype.Timestamptz{}
	}

	return pgtype.Timestamptz{Time: value, Valid: true}
}

func OptionalTimeToPG(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}

	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func requiredFloat(value pgtype.Numeric, field string) (float64, error) {
	if !value.Valid {
		return 0, fmt.Errorf("%s is invalid", field)
	}

	f, err := value.Float64Value()
	if err != nil {
		return 0, fmt.Errorf("%s float conversion: %w", field, err)
	}
	if !f.Valid {
		return 0, fmt.Errorf("%s float conversion is invalid", field)
	}

	return f.Float64, nil
}

func floatToPG(value float64) (pgtype.Numeric, error) {
	var numeric pgtype.Numeric
	if err := numeric.Scan(strconv.FormatFloat(value, 'f', -1, 64)); err != nil {
		return pgtype.Numeric{}, err
	}

	return numeric, nil
}

func optionalInt(value pgtype.Int2) *int {
	if !value.Valid {
		return nil
	}

	v := int(value.Int16)
	return &v
}

func optionalBool(value pgtype.Bool) *bool {
	if !value.Valid {
		return nil
	}

	v := value.Bool
	return &v
}

func optionalIntToPG(value *int) pgtype.Int2 {
	if value == nil {
		return pgtype.Int2{}
	}

	return pgtype.Int2{Int16: int16(*value), Valid: true}
}

func optionalBoolToPG(value *bool) pgtype.Bool {
	if value == nil {
		return pgtype.Bool{}
	}

	return pgtype.Bool{Bool: *value, Valid: true}
}

func optionalInt32ToPG(value *int) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}

	return pgtype.Int4{Int32: int32(*value), Valid: true}
}

func textToPG(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}

	return pgtype.Text{String: value, Valid: true}
}

func textFromPG(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}

	return value.String
}

func intsFromPG(values []int16) []int {
	if len(values) == 0 {
		return []int{}
	}

	out := make([]int, 0, len(values))
	for _, value := range values {
		out = append(out, int(value))
	}

	return out
}

func intsToPG(values []int) []int16 {
	if len(values) == 0 {
		return []int16{}
	}

	out := make([]int16, 0, len(values))
	for _, value := range values {
		out = append(out, int16(value))
	}

	return out
}

func boolsFromPG(values []bool) []bool {
	if len(values) == 0 {
		return []bool{}
	}

	return append([]bool(nil), values...)
}

func boolsToPG(values []bool) []bool {
	if len(values) == 0 {
		return []bool{}
	}

	return append([]bool(nil), values...)
}

func metadataFromBytes(value []byte) (map[string]any, error) {
	if len(value) == 0 {
		return map[string]any{}, nil
	}

	var metadata map[string]any
	if err := json.Unmarshal(value, &metadata); err != nil {
		return nil, err
	}
	if metadata == nil {
		return map[string]any{}, nil
	}

	return metadata, nil
}

func metadataToBytes(value map[string]any) ([]byte, error) {
	if value == nil {
		return []byte("{}"), nil
	}

	return json.Marshal(value)
}
