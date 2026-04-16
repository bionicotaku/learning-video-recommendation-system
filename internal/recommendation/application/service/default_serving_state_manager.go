package service

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	apprepo "learning-video-recommendation-system/internal/recommendation/application/repository"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/mapper"
	recommendationsqlc "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/sqlcgen"
)

type DefaultServingStateManager struct {
	unitRepository  apprepo.UnitServingStateRepository
	videoRepository apprepo.VideoServingStateRepository
	now             func() time.Time
}

var _ ServingStateManager = (*DefaultServingStateManager)(nil)

func NewDefaultServingStateManager(
	unitRepository apprepo.UnitServingStateRepository,
	videoRepository apprepo.VideoServingStateRepository,
) *DefaultServingStateManager {
	return &DefaultServingStateManager{
		unitRepository:  unitRepository,
		videoRepository: videoRepository,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (m *DefaultServingStateManager) ApplySelection(ctx context.Context, runID string, userID string, videos []model.FinalRecommendationItem) error {
	videoIDs := make([]string, 0, len(videos))
	unitIDs := make([]int64, 0, len(videos))
	for _, video := range videos {
		videoIDs = append(videoIDs, video.VideoID)
		unitIDs = appendUniqueInt64(unitIDs, video.CoveredUnits...)
	}

	now := m.now()
	if queries, ok := queriesFromContext(ctx); ok {
		return m.applyWithinQueries(ctx, queries, runID, userID, now, videoIDs, unitIDs)
	}
	return m.applyWithRepositories(ctx, runID, userID, now, videoIDs, unitIDs)
}

func (m *DefaultServingStateManager) applyWithRepositories(ctx context.Context, runID string, userID string, now time.Time, videoIDs []string, unitIDs []int64) error {
	existingUnitStates, err := m.unitRepository.ListByUserAndUnitIDs(ctx, userID, unitIDs)
	if err != nil {
		return err
	}
	existingVideoStates, err := m.videoRepository.ListByUserAndVideoIDs(ctx, userID, videoIDs)
	if err != nil {
		return err
	}

	unitCounts := unitServedCounts(existingUnitStates)
	videoCounts := videoServedCounts(existingVideoStates)
	for _, unitID := range unitIDs {
		if err := m.unitRepository.Upsert(ctx, model.UserUnitServingState{
			UserID:       userID,
			CoarseUnitID: unitID,
			LastServedAt: &now,
			LastRunID:    runID,
			ServedCount:  unitCounts[unitID] + 1,
		}); err != nil {
			return err
		}
	}
	for _, videoID := range videoIDs {
		if err := m.videoRepository.Upsert(ctx, model.UserVideoServingState{
			UserID:       userID,
			VideoID:      videoID,
			LastServedAt: &now,
			LastRunID:    runID,
			ServedCount:  videoCounts[videoID] + 1,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (m *DefaultServingStateManager) applyWithinQueries(ctx context.Context, queries *recommendationsqlc.Queries, runID string, userID string, now time.Time, videoIDs []string, unitIDs []int64) error {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return err
	}
	pgRunID, err := mapper.StringToUUID(runID)
	if err != nil {
		return err
	}

	existingUnitRows, err := queries.ListUserUnitServingStatesByUnitIDs(ctx, recommendationsqlc.ListUserUnitServingStatesByUnitIDsParams{
		UserID:        pgUserID,
		CoarseUnitIds: unitIDs,
	})
	if err != nil {
		return err
	}
	pgVideoIDs := make([]pgtype.UUID, 0, len(videoIDs))
	for _, videoID := range videoIDs {
		pgVideoID, err := mapper.StringToUUID(videoID)
		if err != nil {
			return err
		}
		pgVideoIDs = append(pgVideoIDs, pgVideoID)
	}
	existingVideoRows, err := queries.ListUserVideoServingStatesByVideoIDs(ctx, recommendationsqlc.ListUserVideoServingStatesByVideoIDsParams{
		UserID:   pgUserID,
		VideoIds: pgVideoIDs,
	})
	if err != nil {
		return err
	}

	unitCounts := make(map[int64]int32, len(existingUnitRows))
	for _, row := range existingUnitRows {
		unitCounts[row.CoarseUnitID] = row.ServedCount
	}
	videoCounts := make(map[string]int32, len(existingVideoRows))
	for _, row := range existingVideoRows {
		videoCounts[mapper.UUIDToString(row.VideoID)] = row.ServedCount
	}

	for _, unitID := range unitIDs {
		if err := queries.UpsertUserUnitServingState(ctx, recommendationsqlc.UpsertUserUnitServingStateParams{
			UserID:       pgUserID,
			CoarseUnitID: unitID,
			LastServedAt: mapper.TimePointerToPG(&now),
			LastRunID:    pgRunID,
			ServedCount:  unitCounts[unitID] + 1,
		}); err != nil {
			return err
		}
	}
	for _, videoID := range videoIDs {
		pgVideoID, err := mapper.StringToUUID(videoID)
		if err != nil {
			return err
		}
		if err := queries.UpsertUserVideoServingState(ctx, recommendationsqlc.UpsertUserVideoServingStateParams{
			UserID:       pgUserID,
			VideoID:      pgVideoID,
			LastServedAt: mapper.TimePointerToPG(&now),
			LastRunID:    pgRunID,
			ServedCount:  videoCounts[videoID] + 1,
		}); err != nil {
			return err
		}
	}

	return nil
}

func unitServedCounts(states []model.UserUnitServingState) map[int64]int32 {
	result := make(map[int64]int32, len(states))
	for _, state := range states {
		result[state.CoarseUnitID] = state.ServedCount
	}
	return result
}

func videoServedCounts(states []model.UserVideoServingState) map[string]int32 {
	result := make(map[string]int32, len(states))
	for _, state := range states {
		result[state.VideoID] = state.ServedCount
	}
	return result
}

func appendUniqueInt64(values []int64, additions ...int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, value := range additions {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}
