package pgtext

import "github.com/jackc/pgx/v5/pgtype"

// FromString maps an empty string to an invalid Postgres text value.
func FromString(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

// ToString maps invalid Postgres text values to an empty string.
func ToString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
