package model

import (
	"time"

	"github.com/google/uuid"
)

// UserSchedulerSettings contains per-user scheduler quotas and limits.
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
