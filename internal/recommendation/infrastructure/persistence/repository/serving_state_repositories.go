package repository

import (
	"context"
	"time"

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

func (r *UnitServingStateRepository) IncrementServedCounts(ctx context.Context, userID string, runID string, servedAt time.Time, coarseUnitIDs []int64) error {
	if len(coarseUnitIDs) == 0 {
		return nil
	}

	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return err
	}
	pgRunID, err := mapper.StringToUUID(runID)
	if err != nil {
		return err
	}

	return r.queries.IncrementUserUnitServingStates(ctx, recommendationsqlc.IncrementUserUnitServingStatesParams{
		UserID:        pgUserID,
		LastServedAt:  mapper.TimePointerToPG(&servedAt),
		LastRunID:     pgRunID,
		CoarseUnitIds: uniqueInt64s(coarseUnitIDs),
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

func (r *VideoServingStateRepository) IncrementServedCounts(ctx context.Context, userID string, runID string, servedAt time.Time, videoIDs []string) error {
	if len(videoIDs) == 0 {
		return nil
	}

	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return err
	}
	pgRunID, err := mapper.StringToUUID(runID)
	if err != nil {
		return err
	}

	pgVideoIDs := make([]pgtype.UUID, 0, len(videoIDs))
	for _, videoID := range uniqueStrings(videoIDs) {
		pgVideoID, err := mapper.StringToUUID(videoID)
		if err != nil {
			return err
		}
		pgVideoIDs = append(pgVideoIDs, pgVideoID)
	}

	return r.queries.IncrementUserVideoServingStates(ctx, recommendationsqlc.IncrementUserVideoServingStatesParams{
		UserID:       pgUserID,
		LastServedAt: mapper.TimePointerToPG(&servedAt),
		LastRunID:    pgRunID,
		VideoIds:     pgVideoIDs,
	})
}

func uniqueInt64s(values []int64) []int64 {
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

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
