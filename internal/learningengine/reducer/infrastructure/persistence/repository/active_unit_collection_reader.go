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
