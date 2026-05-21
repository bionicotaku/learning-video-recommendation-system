package repository

import (
	"context"
	"errors"

	apprepo "learning-video-recommendation-system/internal/semantic/application/repository"
	"learning-video-recommendation-system/internal/semantic/domain/model"
	semanticsqlc "learning-video-recommendation-system/internal/semantic/infrastructure/persistence/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UnitCollectionReader struct {
	pool *pgxpool.Pool
}

var _ apprepo.UnitCollectionReader = (*UnitCollectionReader)(nil)

func NewUnitCollectionReader(pool *pgxpool.Pool) *UnitCollectionReader {
	return &UnitCollectionReader{pool: pool}
}

func (r *UnitCollectionReader) ListActiveUnitCollections(ctx context.Context) ([]model.UnitCollection, error) {
	if r.pool == nil {
		return nil, errors.New("pg pool is required")
	}
	rows, err := semanticsqlc.New(r.pool).ListActiveUnitCollections(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]model.UnitCollection, 0, len(rows))
	for _, row := range rows {
		result = append(result, model.UnitCollection{
			CollectionID:    uuidToString(row.CollectionID),
			Slug:            row.Slug,
			Name:            row.Name,
			Description:     textPointer(row.Description),
			Category:        row.Category,
			CoarseUnitCount: row.CoarseUnitCount,
			WordUnitCount:   row.WordUnitCount,
		})
	}
	return result, nil
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
