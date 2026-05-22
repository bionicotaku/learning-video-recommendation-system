package repository

import (
	"context"
	"errors"

	apprepo "learning-video-recommendation-system/internal/learningengine/reducer/application/repository"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
	"learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/mapper"
	learningenginesqlc "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/sqlcgen"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ActiveUnitCollectionReader struct {
	queries *learningenginesqlc.Queries
}

var _ apprepo.ActiveUnitCollectionReader = (*ActiveUnitCollectionReader)(nil)
var _ apprepo.ActiveLearningTargetReader = (*ActiveUnitCollectionReader)(nil)

func NewActiveUnitCollectionReader(pool *pgxpool.Pool) *ActiveUnitCollectionReader {
	return &ActiveUnitCollectionReader{queries: learningenginesqlc.New(pool)}
}

func (r *ActiveUnitCollectionReader) GetActiveUnitCollection(ctx context.Context, userID string) (*model.ActiveUnitCollection, error) {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetActiveUnitCollection(ctx, pgUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &model.ActiveUnitCollection{
		CollectionID:   mapper.UUIDToString(row.ActiveCollectionID),
		CollectionSlug: row.ActiveCollectionSlug,
	}, nil
}

func (r *ActiveUnitCollectionReader) GetActiveLearningTargetCoarseUnitIDs(ctx context.Context, userID string) (model.ActiveLearningTargetCoarseUnitIDs, error) {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return model.ActiveLearningTargetCoarseUnitIDs{}, err
	}
	row, err := r.queries.GetActiveLearningTargetCoarseUnitIDs(ctx, pgUserID)
	if err != nil {
		return model.ActiveLearningTargetCoarseUnitIDs{}, err
	}
	if !row.HasActiveProfile {
		return model.ActiveLearningTargetCoarseUnitIDs{CoarseUnitIDs: []int64{}}, nil
	}
	activeCollection := row.ActiveCollectionSlug
	coarseUnitIDs := row.CoarseUnitIds
	if coarseUnitIDs == nil {
		coarseUnitIDs = []int64{}
	}
	return model.ActiveLearningTargetCoarseUnitIDs{
		ActiveCollection: &activeCollection,
		CoarseUnitIDs:    coarseUnitIDs,
	}, nil
}
