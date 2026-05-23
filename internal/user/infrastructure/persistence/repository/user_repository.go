package repository

import (
	"context"
	"errors"
	"time"

	apprepo "learning-video-recommendation-system/internal/user/application/repository"
	"learning-video-recommendation-system/internal/user/domain/model"
	usersqlc "learning-video-recommendation-system/internal/user/infrastructure/persistence/sqlcgen"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repository struct {
	queries *usersqlc.Queries
}

var _ apprepo.ProfileRepository = (*Repository)(nil)
var _ apprepo.ActivityStatsRepository = (*Repository)(nil)
var _ apprepo.ActivityStatsRecorder = (*Repository)(nil)

func NewRepository(db usersqlc.DBTX) *Repository {
	return &Repository{queries: usersqlc.New(db)}
}

func (r *Repository) GetProfile(ctx context.Context, userID string) (model.UserProfile, bool, error) {
	uuid, err := stringToUUID(userID)
	if err != nil {
		return model.UserProfile{}, false, err
	}
	row, err := r.queries.GetUserProfile(ctx, uuid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.UserProfile{}, false, nil
		}
		return model.UserProfile{}, false, err
	}
	return toUserProfileFromGet(row), true, nil
}

func (r *Repository) RepairProfile(ctx context.Context, userID string) (model.UserProfile, error) {
	uuid, err := stringToUUID(userID)
	if err != nil {
		return model.UserProfile{}, err
	}
	authUser, err := r.queries.GetAuthUser(ctx, uuid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.UserProfile{}, apprepo.ErrAuthUserNotFound
		}
		return model.UserProfile{}, err
	}
	row, err := r.queries.InsertRepairedUserProfile(ctx, usersqlc.InsertRepairedUserProfileParams{
		UserID:           uuid,
		Email:            authUser.Email,
		EmailConfirmedAt: authUser.EmailConfirmedAt,
	})
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return model.UserProfile{}, err
		}
		getRow, err := r.queries.GetUserProfile(ctx, uuid)
		if err != nil {
			return model.UserProfile{}, err
		}
		if err := r.queries.EnsureActivityStats(ctx, uuid); err != nil {
			return model.UserProfile{}, err
		}
		return toUserProfileFromGet(getRow), nil
	}
	if err := r.queries.EnsureActivityStats(ctx, uuid); err != nil {
		return model.UserProfile{}, err
	}
	return toUserProfileFromInsert(row), nil
}

func (r *Repository) UpdateTimezone(ctx context.Context, userID string, timezone string) error {
	uuid, err := stringToUUID(userID)
	if err != nil {
		return err
	}
	return r.queries.UpdateUserTimezone(ctx, usersqlc.UpdateUserTimezoneParams{
		UserID:   uuid,
		Timezone: textValue(&timezone),
	})
}

func (r *Repository) UpdateOnboardingStatus(ctx context.Context, userID string, status string) error {
	uuid, err := stringToUUID(userID)
	if err != nil {
		return err
	}
	return r.queries.UpdateOnboardingStatus(ctx, usersqlc.UpdateOnboardingStatusParams{
		UserID:           uuid,
		OnboardingStatus: status,
	})
}

func (r *Repository) EnsureActivityStats(ctx context.Context, userID string) error {
	uuid, err := stringToUUID(userID)
	if err != nil {
		return err
	}
	return r.queries.EnsureActivityStats(ctx, uuid)
}

func (r *Repository) GetActivityStats(ctx context.Context, userID string) (model.ActivityStats, bool, error) {
	uuid, err := stringToUUID(userID)
	if err != nil {
		return model.ActivityStats{}, false, err
	}
	row, err := r.queries.GetActivityStats(ctx, uuid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ActivityStats{}, false, nil
		}
		return model.ActivityStats{}, false, err
	}
	return toActivityStats(row), true, nil
}

func (r *Repository) ListDailyActivityStats(ctx context.Context, userID string, fromDate time.Time, toDate time.Time) ([]model.DailyActivityStats, error) {
	uuid, err := stringToUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.ListDailyActivityStats(ctx, usersqlc.ListDailyActivityStatsParams{
		UserID:   uuid,
		FromDate: dateValue(fromDate),
		ToDate:   dateValue(toDate),
	})
	if err != nil {
		return nil, err
	}
	result := make([]model.DailyActivityStats, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDailyActivityStats(row))
	}
	return result, nil
}

func (r *Repository) GetCurrentActivityStreakDays(ctx context.Context, userID string, today time.Time) (int64, error) {
	uuid, err := stringToUUID(userID)
	if err != nil {
		return 0, err
	}
	return r.queries.GetCurrentActivityStreakDays(ctx, usersqlc.GetCurrentActivityStreakDaysParams{
		UserID: uuid,
		Today:  dateValue(today),
	})
}

func (r *Repository) AddWatchDuration(ctx context.Context, userID string, activityAt time.Time, deltaWatchMS int64) error {
	if deltaWatchMS <= 0 {
		return nil
	}
	params, err := r.dailyParams(ctx, userID, activityAt)
	if err != nil {
		return err
	}
	return r.queries.AddWatchDuration(ctx, usersqlc.AddWatchDurationParams{
		UserID:       params.userID,
		LocalDate:    params.localDate,
		Timezone:     params.timezone,
		DeltaWatchMs: deltaWatchMS,
		ActivityAt:   timestamptzValue(activityAt),
	})
}

func (r *Repository) IncrementQuizAttempt(ctx context.Context, userID string, completedAt time.Time) error {
	params, err := r.dailyParams(ctx, userID, completedAt)
	if err != nil {
		return err
	}
	return r.queries.IncrementQuizAttempt(ctx, usersqlc.IncrementQuizAttemptParams{
		UserID:     params.userID,
		LocalDate:  params.localDate,
		Timezone:   params.timezone,
		ActivityAt: timestamptzValue(completedAt),
	})
}

func (r *Repository) IncrementStartedUnit(ctx context.Context, userID string) error {
	uuid, err := stringToUUID(userID)
	if err != nil {
		return err
	}
	return r.queries.IncrementStartedUnit(ctx, uuid)
}

func (r *Repository) IncrementLearningInteraction(ctx context.Context, userID string, occurredAt time.Time) error {
	params, err := r.dailyParams(ctx, userID, occurredAt)
	if err != nil {
		return err
	}
	return r.queries.IncrementLearningInteraction(ctx, usersqlc.IncrementLearningInteractionParams{
		UserID:     params.userID,
		LocalDate:  params.localDate,
		Timezone:   params.timezone,
		ActivityAt: timestamptzValue(occurredAt),
	})
}

type dailyParamSet struct {
	userID    pgtype.UUID
	localDate pgtype.Date
	timezone  string
}

func (r *Repository) dailyParams(ctx context.Context, userID string, activityAt time.Time) (dailyParamSet, error) {
	uuid, err := stringToUUID(userID)
	if err != nil {
		return dailyParamSet{}, err
	}
	timezone := model.DefaultTimezone
	if profile, found, err := r.GetProfile(ctx, userID); err != nil {
		return dailyParamSet{}, err
	} else if found && profile.Timezone != nil && *profile.Timezone != "" {
		timezone = *profile.Timezone
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		timezone = model.DefaultTimezone
		location, _ = time.LoadLocation(timezone)
	}
	local := activityAt.UTC().In(location)
	return dailyParamSet{
		userID:    uuid,
		localDate: dateValue(local),
		timezone:  timezone,
	}, nil
}

func toUserProfileFromGet(row usersqlc.GetUserProfileRow) model.UserProfile {
	return model.UserProfile{
		UserID:           uuidToString(row.UserID),
		Email:            textPointer(row.Email),
		EmailConfirmedAt: timestamptzPointer(row.EmailConfirmedAt),
		DisplayName:      row.DisplayName,
		AvatarURL:        textPointer(row.AvatarUrl),
		Locale:           row.Locale,
		Timezone:         textPointer(row.Timezone),
		OnboardingStatus: row.OnboardingStatus,
		BirthDate:        datePointer(row.BirthDate),
		Gender:           textPointer(row.Gender),
		EducationStage:   textPointer(row.EducationStage),
		IPRegion:         textPointer(row.IpRegion),
		CreatedAt:        timeOrZero(row.CreatedAt),
		UpdatedAt:        timeOrZero(row.UpdatedAt),
	}
}

func toUserProfileFromInsert(row usersqlc.InsertRepairedUserProfileRow) model.UserProfile {
	return model.UserProfile{
		UserID:           uuidToString(row.UserID),
		Email:            textPointer(row.Email),
		EmailConfirmedAt: timestamptzPointer(row.EmailConfirmedAt),
		DisplayName:      row.DisplayName,
		AvatarURL:        textPointer(row.AvatarUrl),
		Locale:           row.Locale,
		Timezone:         textPointer(row.Timezone),
		OnboardingStatus: row.OnboardingStatus,
		BirthDate:        datePointer(row.BirthDate),
		Gender:           textPointer(row.Gender),
		EducationStage:   textPointer(row.EducationStage),
		IPRegion:         textPointer(row.IpRegion),
		CreatedAt:        timeOrZero(row.CreatedAt),
		UpdatedAt:        timeOrZero(row.UpdatedAt),
	}
}

func toActivityStats(row usersqlc.AppUserUserActivityStat) model.ActivityStats {
	return model.ActivityStats{
		UserID:           uuidToString(row.UserID),
		TotalWatchMS:     row.TotalWatchMs,
		QuizAttemptCount: row.QuizAttemptCount,
		StartedUnitCount: row.StartedUnitCount,
		UpdatedAt:        timeOrZero(row.UpdatedAt),
	}
}

func toDailyActivityStats(row usersqlc.AppUserUserDailyActivityStat) model.DailyActivityStats {
	return model.DailyActivityStats{
		UserID:                   uuidToString(row.UserID),
		LocalDate:                timeOrZeroDate(row.LocalDate),
		Timezone:                 row.Timezone,
		WatchMS:                  row.WatchMs,
		QuizAttemptCount:         row.QuizAttemptCount,
		LearningInteractionCount: row.LearningInteractionCount,
		FirstActivityAt:          timestamptzPointer(row.FirstActivityAt),
		LastActivityAt:           timestamptzPointer(row.LastActivityAt),
		UpdatedAt:                timeOrZero(row.UpdatedAt),
	}
}

func timeOrZero(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time.UTC()
}

func timeOrZeroDate(value pgtype.Date) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return time.Date(value.Time.Year(), value.Time.Month(), value.Time.Day(), 0, 0, 0, 0, time.UTC)
}

func datePointer(value pgtype.Date) *time.Time {
	if !value.Valid {
		return nil
	}
	result := timeOrZeroDate(value)
	return &result
}
