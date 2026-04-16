package repository

import (
	"context"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/mapper"
	recommendationsqlc "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/sqlcgen"
)

type RecommendableVideoUnitReader struct {
	queries *recommendationsqlc.Queries
}

var _ apprepo.RecommendableVideoUnitReader = (*RecommendableVideoUnitReader)(nil)

func NewRecommendableVideoUnitReader(db recommendationsqlc.DBTX) *RecommendableVideoUnitReader {
	return &RecommendableVideoUnitReader{
		queries: recommendationsqlc.New(db),
	}
}

func (r *RecommendableVideoUnitReader) ListByUnitIDs(ctx context.Context, coarseUnitIDs []int64) ([]model.RecommendableVideoUnit, error) {
	rows, err := r.queries.ListRecommendableVideoUnitsByUnitIDs(ctx, coarseUnitIDs)
	if err != nil {
		return nil, err
	}

	result := make([]model.RecommendableVideoUnit, 0, len(rows))
	for _, row := range rows {
		mapped, err := mapper.ToRecommendableVideoUnit(row)
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}
	return result, nil
}
