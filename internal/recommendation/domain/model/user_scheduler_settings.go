package model

import (
	"time"

	"github.com/google/uuid"
)

// UserSchedulerSettings contains per-user recommendation quotas and limits.
type UserSchedulerSettings struct {
	UserID               uuid.UUID
	SessionDefaultLimit  int
	DailyNewUnitQuota    int
	DailyReviewSoftLimit int
	DailyReviewHardLimit int
	Timezone             string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// DefaultUserSchedulerSettings returns the MVP recommendation defaults.
func DefaultUserSchedulerSettings() UserSchedulerSettings {
	return UserSchedulerSettings{
		SessionDefaultLimit:  20,
		DailyNewUnitQuota:    8,
		DailyReviewSoftLimit: 30,
		DailyReviewHardLimit: 60,
	}
}
