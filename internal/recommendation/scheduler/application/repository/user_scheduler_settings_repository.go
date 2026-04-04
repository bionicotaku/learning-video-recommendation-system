package repository

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"

	"github.com/google/uuid"
)

type UserSchedulerSettingsRepository interface {
	GetOrDefault(ctx context.Context, q sqlcgen.Querier, userID uuid.UUID) (*model.UserSchedulerSettings, error)
}
