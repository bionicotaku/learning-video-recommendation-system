package repository

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"

	"github.com/google/uuid"
)

type UserSchedulerSettingsRepository interface {
	GetOrDefault(ctx context.Context, userID uuid.UUID) (*model.UserSchedulerSettings, error)
}
