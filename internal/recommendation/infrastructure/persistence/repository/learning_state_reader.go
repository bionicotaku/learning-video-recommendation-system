package repository

import (
	"context"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/mapper"
	recommendationsqlc "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/sqlcgen"
)

type LearningStateReader struct {
	queries *recommendationsqlc.Queries
}

var _ apprepo.LearningStateReader = (*LearningStateReader)(nil)

func NewLearningStateReader(db recommendationsqlc.DBTX) *LearningStateReader {
	return &LearningStateReader{
		queries: recommendationsqlc.New(db),
	}
}

func (r *LearningStateReader) ListActiveByUser(ctx context.Context, userID string) ([]model.LearningStateSnapshot, error) {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.ListLearningStatesForRecommendation(ctx, pgUserID)
	if err != nil {
		return nil, err
	}

	result := make([]model.LearningStateSnapshot, 0, len(rows))
	for _, row := range rows {
		mapped, err := mapper.ToLearningStateSnapshot(row)
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}
	return result, nil
}
