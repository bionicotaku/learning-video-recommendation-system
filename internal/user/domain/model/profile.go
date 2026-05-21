package model

import "time"

const (
	OnboardingStatusNew                = "new"
	OnboardingStatusCollectionSelected = "collection_selected"
	OnboardingStatusCompleted          = "completed"
	DefaultLocale                      = "zh-CN"
	DefaultTimezone                    = "UTC"
)

type UserProfile struct {
	UserID           string
	Email            *string
	EmailConfirmedAt *time.Time
	DisplayName      *string
	AvatarURL        *string
	Locale           string
	Timezone         *string
	OnboardingStatus string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ActivityStats struct {
	UserID           string
	TotalWatchMS     int64
	QuizAttemptCount int64
	StartedUnitCount int64
	UpdatedAt        time.Time
}

type DailyActivityStats struct {
	UserID                   string
	LocalDate                time.Time
	Timezone                 string
	WatchMS                  int64
	QuizAttemptCount         int64
	LearningInteractionCount int64
	FirstActivityAt          *time.Time
	LastActivityAt           *time.Time
	UpdatedAt                time.Time
}
