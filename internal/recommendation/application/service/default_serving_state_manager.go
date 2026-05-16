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
		videoIDs = appendUniqueString(videoIDs, video.VideoID)
		unitIDs = appendUniqueInt64(unitIDs, model.LearningUnitIDs(video.LearningUnits)...)
	}

	now := m.now().UTC()
	if queries, ok := queriesFromContext(ctx); ok {
		return m.applyWithinQueries(ctx, queries, runID, userID, now, videoIDs, unitIDs)
	}
	return m.applyWithRepositories(ctx, runID, userID, now, videoIDs, unitIDs)
}

func (m *DefaultServingStateManager) applyWithRepositories(ctx context.Context, runID string, userID string, now time.Time, videoIDs []string, unitIDs []int64) error {
	if len(unitIDs) > 0 {
		if err := m.unitRepository.IncrementServedCounts(ctx, userID, runID, now, unitIDs); err != nil {
			return err
		}
	}
	if len(videoIDs) > 0 {
		if err := m.videoRepository.IncrementServedCounts(ctx, userID, runID, now, videoIDs); err != nil {
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

	pgVideoIDs := make([]pgtype.UUID, 0, len(videoIDs))
	for _, videoID := range videoIDs {
		pgVideoID, err := mapper.StringToUUID(videoID)
		if err != nil {
			return err
		}
		pgVideoIDs = append(pgVideoIDs, pgVideoID)
	}

	if len(unitIDs) > 0 {
		if err := queries.IncrementUserUnitServingStates(ctx, recommendationsqlc.IncrementUserUnitServingStatesParams{
			UserID:        pgUserID,
			LastServedAt:  mapper.TimePointerToPG(&now),
			LastRunID:     pgRunID,
			CoarseUnitIds: unitIDs,
		}); err != nil {
			return err
		}
	}

	if len(pgVideoIDs) > 0 {
		if err := queries.IncrementUserVideoServingStates(ctx, recommendationsqlc.IncrementUserVideoServingStatesParams{
			UserID:       pgUserID,
			LastServedAt: mapper.TimePointerToPG(&now),
			LastRunID:    pgRunID,
			VideoIds:     pgVideoIDs,
		}); err != nil {
			return err
		}
	}

	return nil
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

func appendUniqueString(values []string, additions ...string) []string {
	seen := make(map[string]struct{}, len(values))
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
