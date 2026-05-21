package repository

import (
	"context"
	"errors"
	"time"

	"learning-video-recommendation-system/internal/user/domain/model"
)

var ErrAuthUserNotFound = errors.New("auth user not found")

type ProfileRepository interface {
	GetProfile(ctx context.Context, userID string) (model.UserProfile, bool, error)
	RepairProfile(ctx context.Context, userID string) (model.UserProfile, error)
	UpdateTimezone(ctx context.Context, userID string, timezone string) error
	UpdateOnboardingStatus(ctx context.Context, userID string, status string) error
}

type ActivityStatsRepository interface {
	EnsureActivityStats(ctx context.Context, userID string) error
	GetActivityStats(ctx context.Context, userID string) (model.ActivityStats, bool, error)
	ListDailyActivityStats(ctx context.Context, userID string, fromDate time.Time, toDate time.Time) ([]model.DailyActivityStats, error)
	GetCurrentActivityStreakDays(ctx context.Context, userID string, today time.Time) (int64, error)
}

type ActivityStatsRecorder interface {
	AddWatchDuration(ctx context.Context, userID string, activityAt time.Time, deltaWatchMS int64) error
	IncrementQuizAttempt(ctx context.Context, userID string, completedAt time.Time) error
	IncrementStartedUnit(ctx context.Context, userID string) error
	IncrementLearningInteraction(ctx context.Context, userID string, occurredAt time.Time) error
}
