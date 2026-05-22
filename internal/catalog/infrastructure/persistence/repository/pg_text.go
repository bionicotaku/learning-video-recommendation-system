package repository

import "github.com/jackc/pgx/v5/pgtype"

func textPointer(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	text := value.String
	return &text
}
