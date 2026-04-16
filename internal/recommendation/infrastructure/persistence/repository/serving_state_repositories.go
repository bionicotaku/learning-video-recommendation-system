package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/mapper"
	recommendationsqlc "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/sqlcgen"
)

type UnitServingStateRepository struct {
	queries *recommendationsqlc.Queries
}

type VideoServingStateRepository struct {
	queries *recommendationsqlc.Queries
}

var _ apprepo.UnitServingStateRepository = (*UnitServingStateRepository)(nil)
var _ apprepo.VideoServingStateRepository = (*VideoServingStateRepository)(nil)

func NewUnitServingStateRepository(db recommendationsqlc.DBTX) *UnitServingStateRepository {
	return &UnitServingStateRepository{queries: recommendationsqlc.New(db)}
}

func NewVideoServingStateRepository(db recommendationsqlc.DBTX) *VideoServingStateRepository {
	return &VideoServingStateRepository{queries: recommendationsqlc.New(db)}
}

func (r *UnitServingStateRepository) ListByUserAndUnitIDs(ctx context.Context, userID string, coarseUnitIDs []int64) ([]model.UserUnitServingState, error) {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.ListUserUnitServingStatesByUnitIDs(ctx, recommendationsqlc.ListUserUnitServingStatesByUnitIDsParams{
		UserID:        pgUserID,
		CoarseUnitIds: coarseUnitIDs,
	})
	if err != nil {
		return nil, err
	}

	result := make([]model.UserUnitServingState, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapper.ToUserUnitServingState(row))
	}
	return result, nil
}

func (r *UnitServingStateRepository) Upsert(ctx context.Context, state model.UserUnitServingState) error {
	pgUserID, err := mapper.StringToUUID(state.UserID)
	if err != nil {
		return err
	}
	pgRunID, err := mapper.StringToUUID(state.LastRunID)
	if err != nil {
		return err
	}

	return r.queries.UpsertUserUnitServingState(ctx, recommendationsqlc.UpsertUserUnitServingStateParams{
		UserID:       pgUserID,
		CoarseUnitID: state.CoarseUnitID,
		LastServedAt: mapper.TimePointerToPG(state.LastServedAt),
		LastRunID:    pgRunID,
		ServedCount:  state.ServedCount,
	})
}

func (r *VideoServingStateRepository) ListByUserAndVideoIDs(ctx context.Context, userID string, videoIDs []string) ([]model.UserVideoServingState, error) {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return nil, err
	}

	pgVideoIDs := make([]pgtype.UUID, 0, len(videoIDs))
	for _, videoID := range videoIDs {
		pgVideoID, err := mapper.StringToUUID(videoID)
		if err != nil {
			return nil, err
		}
		pgVideoIDs = append(pgVideoIDs, pgVideoID)
	}

	rows, err := r.queries.ListUserVideoServingStatesByVideoIDs(ctx, recommendationsqlc.ListUserVideoServingStatesByVideoIDsParams{
		UserID:   pgUserID,
		VideoIds: pgVideoIDs,
	})
	if err != nil {
		return nil, err
	}

	result := make([]model.UserVideoServingState, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapper.ToUserVideoServingState(row))
	}
	return result, nil
}

func (r *VideoServingStateRepository) Upsert(ctx context.Context, state model.UserVideoServingState) error {
	pgUserID, err := mapper.StringToUUID(state.UserID)
	if err != nil {
		return err
	}
	pgVideoID, err := mapper.StringToUUID(state.VideoID)
	if err != nil {
		return err
	}
	pgRunID, err := mapper.StringToUUID(state.LastRunID)
	if err != nil {
		return err
	}

	return r.queries.UpsertUserVideoServingState(ctx, recommendationsqlc.UpsertUserVideoServingStateParams{
		UserID:       pgUserID,
		VideoID:      pgVideoID,
		LastServedAt: mapper.TimePointerToPG(state.LastServedAt),
		LastRunID:    pgRunID,
		ServedCount:  state.ServedCount,
	})
}
