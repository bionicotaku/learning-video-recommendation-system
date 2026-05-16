package pguuid

import "github.com/jackc/pgx/v5/pgtype"

// FromString maps an empty string to an invalid UUID and parses non-empty UUID strings.
func FromString(value string) (pgtype.UUID, error) {
	if value == "" {
		return pgtype.UUID{}, nil
	}

	var result pgtype.UUID
	if err := result.Scan(value); err != nil {
		return pgtype.UUID{}, err
	}
	return result, nil
}

// ToString maps invalid UUID values to an empty string.
func ToString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return value.String()
}

// SliceFromStrings parses a slice of UUID strings.
func SliceFromStrings(values []string) ([]pgtype.UUID, error) {
	uuids := make([]pgtype.UUID, 0, len(values))
	for _, value := range values {
		uuid, err := FromString(value)
		if err != nil {
			return nil, err
		}
		uuids = append(uuids, uuid)
	}
	return uuids, nil
}
