package repository

import (
	"context"
	"errors"

	apprepo "learning-video-recommendation-system/internal/recommendation/scheduler/application/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/mapper"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type userSchedulerSettingsRepository struct{}

func NewUserSchedulerSettingsRepository() apprepo.UserSchedulerSettingsRepository {
	return userSchedulerSettingsRepository{}
}

func (userSchedulerSettingsRepository) GetOrDefault(ctx context.Context, q sqlcgen.Querier, userID uuid.UUID) (*model.UserSchedulerSettings, error) {
	row, err := q.GetUserSchedulerSettings(ctx, mapper.UUIDToPG(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &model.UserSchedulerSettings{
				UserID:               userID,
				SessionDefaultLimit:  20,
				DailyNewUnitQuota:    8,
				DailyReviewSoftLimit: 30,
				DailyReviewHardLimit: 60,
			}, nil
		}
		return nil, err
	}

	settings, err := mapper.UserSchedulerSettingsFromRow(row)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}
