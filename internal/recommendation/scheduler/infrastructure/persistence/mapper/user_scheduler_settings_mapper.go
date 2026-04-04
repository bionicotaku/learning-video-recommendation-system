package mapper

import (
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"
)

// UserSchedulerSettingsFromRow maps a sqlc settings row to the domain model.
func UserSchedulerSettingsFromRow(row sqlcgen.LearningUserSchedulerSetting) (model.UserSchedulerSettings, error) {
	userID, err := requiredUUID(row.UserID, "user_scheduler_settings.user_id")
	if err != nil {
		return model.UserSchedulerSettings{}, err
	}
	createdAt, err := requiredTime(row.CreatedAt, "user_scheduler_settings.created_at")
	if err != nil {
		return model.UserSchedulerSettings{}, err
	}
	updatedAt, err := requiredTime(row.UpdatedAt, "user_scheduler_settings.updated_at")
	if err != nil {
		return model.UserSchedulerSettings{}, err
	}

	return model.UserSchedulerSettings{
		UserID:               userID,
		SessionDefaultLimit:  int(row.SessionDefaultLimit),
		DailyNewUnitQuota:    int(row.DailyNewUnitQuota),
		DailyReviewSoftLimit: int(row.DailyReviewSoftLimit),
		DailyReviewHardLimit: int(row.DailyReviewHardLimit),
		Timezone:             textFromPG(row.Timezone),
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAt,
	}, nil
}
