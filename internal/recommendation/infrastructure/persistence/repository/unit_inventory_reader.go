package repository

import (
	"context"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/mapper"
	recommendationsqlc "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/sqlcgen"
)

type UnitInventoryReader struct {
	queries *recommendationsqlc.Queries
}

var _ apprepo.UnitInventoryReader = (*UnitInventoryReader)(nil)

func NewUnitInventoryReader(db recommendationsqlc.DBTX) *UnitInventoryReader {
	return &UnitInventoryReader{
		queries: recommendationsqlc.New(db),
	}
}

func (r *UnitInventoryReader) ListByUnitIDs(ctx context.Context, coarseUnitIDs []int64) ([]model.UnitVideoInventory, error) {
	rows, err := r.queries.ListUnitVideoInventoryByUnitIDs(ctx, coarseUnitIDs)
	if err != nil {
		return nil, err
	}

	result := make([]model.UnitVideoInventory, 0, len(rows))
	for _, row := range rows {
		mapped, err := mapper.ToUnitVideoInventory(row)
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}
	return result, nil
}
