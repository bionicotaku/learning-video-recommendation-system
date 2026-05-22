package repository

import (
	"context"
	"errors"

	apprepo "learning-video-recommendation-system/internal/catalog/application/repository"
	"learning-video-recommendation-system/internal/catalog/domain/model"
	catalogsqlc "learning-video-recommendation-system/internal/catalog/infrastructure/persistence/sqlcgen"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UnitLabelReader struct {
	pool *pgxpool.Pool
}

var _ apprepo.UnitLabelReader = (*UnitLabelReader)(nil)

func NewUnitLabelReader(pool *pgxpool.Pool) *UnitLabelReader {
	return &UnitLabelReader{pool: pool}
}

func (r *UnitLabelReader) ListUnitLabelsByIDs(ctx context.Context, coarseUnitIDs []int64) ([]model.UnitLabel, error) {
	if r.pool == nil {
		return nil, errors.New("pg pool is required")
	}
	if len(coarseUnitIDs) == 0 {
		return nil, nil
	}

	rows, err := catalogsqlc.New(r.pool).ListUnitLabelsByIDs(ctx, coarseUnitIDs)
	if err != nil {
		return nil, err
	}
	result := make([]model.UnitLabel, 0, len(rows))
	for _, row := range rows {
		result = append(result, model.UnitLabel{
			CoarseUnitID: row.ID,
			Text:         row.Label,
		})
	}
	return result, nil
}
