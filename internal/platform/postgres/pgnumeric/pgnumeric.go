package pgnumeric

import (
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
)

// FromFloat64 maps a float64 to a Postgres numeric value without applying domain rounding.
func FromFloat64(value float64) (pgtype.Numeric, error) {
	var result pgtype.Numeric
	if err := result.Scan(strconv.FormatFloat(value, 'f', -1, 64)); err != nil {
		return pgtype.Numeric{}, err
	}
	return result, nil
}

// ToFloat64 maps an invalid Postgres numeric value to 0.
func ToFloat64(value pgtype.Numeric) (float64, error) {
	floatValue, err := value.Float64Value()
	if err != nil {
		return 0, err
	}
	if !floatValue.Valid {
		return 0, nil
	}
	return floatValue.Float64, nil
}
