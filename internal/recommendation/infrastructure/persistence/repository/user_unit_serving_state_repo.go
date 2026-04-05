package repository

import (
	"context"
	"time"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/mapper"
	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/sqlcgen"

	"github.com/google/uuid"
)

type userUnitServingStateRepository struct {
	querier sqlcgen.Querier
}

func NewUserUnitServingStateRepository(querier sqlcgen.Querier) apprepo.UserUnitServingStateRepository {
	return userUnitServingStateRepository{querier: querier}
}

func (r userUnitServingStateRepository) TouchRecommendedAt(ctx context.Context, userID uuid.UUID, runID uuid.UUID, coarseUnitIDs []int64, recommendedAt time.Time) error {
	q, err := resolveQuerier(ctx, r.querier)
	if err != nil {
		return err
	}

	for _, coarseUnitID := range uniqueInt64s(coarseUnitIDs) {
		if err := q.UpsertUserUnitServingState(ctx, mapper.UserUnitServingStateToUpsertParams(userID, coarseUnitID, runID, recommendedAt)); err != nil {
			return err
		}
	}

	return nil
}

func uniqueInt64s(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}
